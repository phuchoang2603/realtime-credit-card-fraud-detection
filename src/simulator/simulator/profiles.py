from dataclasses import dataclass
from datetime import date
from hashlib import sha256

import numpy as np

from simulator.config import HistoricalConfig
from simulator.identity import entity_id, fingerprint

CATEGORIES = ("phone", "laptop", "accessory", "tablet", "headphones", "smartwatch", "game_console", "camera")
CATEGORY_PRICES = np.array([35000, 65000, 3000, 30000, 12000, 18000, 28000, 45000])
CONDITIONS = ("excellent", "good", "fair")
CONDITION_WEIGHTS = np.array([0.4, 0.4, 0.2])
CONDITION_PRICE_FACTORS = np.array([1.0, 0.88, 0.75])
ACCESSORY = CATEGORIES.index("accessory")
COUNTRIES = ("US", "CA", "GB")
COUNTRY_WEIGHTS = np.array([0.8, 0.12, 0.08])
REGIONS = {"US": ("NY", "10001"), "CA": ("ON", "M5V 2T6"), "GB": ("LDN", "EC1A 1BB")}

LEGITIMATE = 0
OBVIOUS_HIGH_VALUE = 1
STOLEN_CARD_NEW_DEVICE = 2
ACCOUNT_TAKEOVER = 3
CARD_TESTING = 4
SCENARIO_NAMES = ("legitimate", "obvious_high_value", "stolen_card_new_device", "account_takeover", "card_testing")

PROFILE_STAGE = 1
SCHEDULE_STAGE = 2
COMPROMISE_STAGE = 3
CARD_TESTING_STAGE = 4
HISTORY_STAGE = 5
DUPLICATE_STAGE = 6


def stage_rng(seed: int, stage: int, *keys: int) -> np.random.Generator:
    return np.random.default_rng(np.random.SeedSequence([seed, stage, *keys]))


def zipf_weights(count: int, exponent: float) -> np.ndarray:
    weights = np.arange(1, count + 1, dtype=float) ** (-exponent)
    return weights / weights.sum()


@dataclass(frozen=True)
class Profiles:
    merchant_ids: tuple[str, ...]
    merchant_external_ids: tuple[str, ...]
    merchant_categories: np.ndarray
    merchant_age_days: np.ndarray
    customer_ids: tuple[str, ...]
    customer_external_ids: tuple[str, ...]
    customer_age_days: np.ndarray
    customer_countries: np.ndarray
    familiar_merchants: np.ndarray
    familiar_devices: np.ndarray
    home_networks: np.ndarray
    addresses: np.ndarray
    payment_methods: np.ndarray
    device_pool: tuple[str, ...]
    network_pool: tuple[str, ...]
    address_pool: tuple[str, ...]
    method_pool: tuple[str, ...]

    def digest(self) -> str:
        digest = sha256()
        for name, value in sorted(vars(self).items()):
            digest.update(name.encode())
            if isinstance(value, np.ndarray):
                digest.update(np.ascontiguousarray(value, dtype=np.int64).tobytes())
            else:
                digest.update("\x1f".join(value).encode())
        return digest.hexdigest()


@dataclass(frozen=True)
class Schedule:
    customers: np.ndarray
    merchants: np.ndarray
    seconds: np.ndarray
    scenarios: np.ndarray
    groups: np.ndarray

    def __len__(self) -> int:
        return len(self.customers)


def build_profiles(config: HistoricalConfig) -> Profiles:
    counts = config.counts
    random = stage_rng(config.seed, PROFILE_STAGE)
    key = config.synthetic_key
    category_weights = zipf_weights(len(CATEGORIES), config.behavior.category_zipf_exponent)
    merchant_weights = zipf_weights(counts.merchants, config.behavior.merchant_zipf_exponent)
    merchant_external = tuple(f"seller-{index + 1:06d}" for index in range(counts.merchants))
    customer_external = tuple(f"buyer-{index + 1:06d}" for index in range(counts.customers))
    familiar_count = min(3, counts.merchants)
    return Profiles(
        merchant_ids=tuple(entity_id(config.integration_id, "merchant", value) for value in merchant_external),
        merchant_external_ids=merchant_external,
        merchant_categories=random.choice(len(CATEGORIES), size=counts.merchants, p=category_weights),
        merchant_age_days=random.integers(30, 1500, size=counts.merchants),
        customer_ids=tuple(entity_id(config.integration_id, "customer", value) for value in customer_external),
        customer_external_ids=customer_external,
        customer_age_days=random.integers(1, 2500, size=counts.customers),
        customer_countries=random.choice(len(COUNTRIES), size=counts.customers, p=COUNTRY_WEIGHTS),
        familiar_merchants=np.stack(
            [
                random.choice(counts.merchants, size=familiar_count, replace=False, p=merchant_weights)
                for _ in range(counts.customers)
            ]
        ),
        familiar_devices=random.integers(0, counts.devices, size=(counts.customers, 2)),
        home_networks=random.integers(0, counts.ip_networks, size=counts.customers),
        addresses=random.integers(0, counts.addresses, size=counts.customers),
        payment_methods=random.integers(0, counts.payment_methods, size=(counts.customers, 2)),
        device_pool=tuple(fingerprint(key, "device", f"device-{index:06d}") for index in range(counts.devices)),
        network_pool=tuple(
            fingerprint(key, "ip_network", f"10.{index // 256}.{index % 256}.1") for index in range(counts.ip_networks)
        ),
        address_pool=tuple(
            fingerprint(key, "address", f"{index} Market Street, Example City") for index in range(counts.addresses)
        ),
        method_pool=tuple(
            fingerprint(key, "payment_method", f"tok_card-{index:06d}_ok") for index in range(counts.payment_methods)
        ),
    )


def compromised_on(config: HistoricalConfig, scenario: int, offset: int) -> np.ndarray:
    settings = (
        config.scenarios.stolen_card_new_device
        if scenario == STOLEN_CARD_NEW_DEVICE
        else config.scenarios.account_takeover
    )
    random = stage_rng(config.seed, COMPROMISE_STAGE, scenario, offset)
    return np.flatnonzero(random.random(config.counts.customers) < settings.rate)


def active_compromises(config: HistoricalConfig, scenario: int, offset: int) -> np.ndarray:
    """Customers whose compromise window covers this day, recovered without replaying purchases."""
    settings = (
        config.scenarios.stolen_card_new_device
        if scenario == STOLEN_CARD_NEW_DEVICE
        else config.scenarios.account_takeover
    )
    starts = [
        np.column_stack([selected, np.full(len(selected), start)])
        for start in range(max(0, offset - settings.window_days + 1), offset + 1)
        if len(selected := compromised_on(config, scenario, start))
    ]
    if not starts:
        return np.empty((0, 2), dtype=np.int64)
    return np.concatenate(starts)


def schedule_for_day(config: HistoricalConfig, day: date, profiles: Profiles) -> Schedule:
    offset = (day - config.start_date).days
    if not 0 <= offset < config.duration_days:
        raise ValueError("day is outside the configured dataset")
    behavior = config.behavior
    scenarios = config.scenarios
    random = stage_rng(config.seed, SCHEDULE_STAGE, offset)
    customer_count = len(profiles.customer_ids)
    merchant_weights = zipf_weights(len(profiles.merchant_ids), behavior.merchant_zipf_exponent)

    customers = np.repeat(
        np.arange(customer_count), random.poisson(behavior.purchases_per_customer_day, customer_count)
    )
    familiar = random.random(len(customers)) < behavior.familiar_merchant_probability
    familiar_choice = profiles.familiar_merchants[
        customers, random.integers(0, profiles.familiar_merchants.shape[1], len(customers))
    ]
    merchants = np.where(
        familiar, familiar_choice, random.choice(len(merchant_weights), len(customers), p=merchant_weights)
    )
    seconds = random.integers(0, 86400, len(customers))
    kinds = np.where(random.random(len(customers)) < scenarios.obvious_high_value.rate, OBVIOUS_HIGH_VALUE, LEGITIMATE)
    groups = np.full(len(customers), -1)
    parts = [(customers, merchants, seconds, kinds, groups)]

    for scenario, settings in (
        (STOLEN_CARD_NEW_DEVICE, scenarios.stolen_card_new_device),
        (ACCOUNT_TAKEOVER, scenarios.account_takeover),
    ):
        active = active_compromises(config, scenario, offset)
        attempts = random.poisson(settings.attempts_per_day, len(active))
        victims = np.repeat(active[:, 0], attempts)
        parts.append(
            (
                victims,
                profiles.familiar_merchants[victims, 0],
                random.integers(0, 86400, len(victims)),
                np.full(len(victims), scenario),
                np.repeat(active[:, 1], attempts),
            )
        )

    testing = scenarios.card_testing
    bursts = random.poisson(testing.rate)
    for burst in range(bursts):
        start = int(random.integers(0, 86400 - testing.burst_minutes * 60))
        parts.append(
            (
                np.full(testing.burst_size, int(random.integers(0, customer_count))),
                np.full(testing.burst_size, int(random.choice(len(merchant_weights), p=merchant_weights))),
                np.sort(start + random.integers(0, testing.burst_minutes * 60, testing.burst_size)),
                np.full(testing.burst_size, CARD_TESTING),
                np.full(testing.burst_size, burst),
            )
        )

    customers, merchants, seconds, kinds, groups = (np.concatenate(column) for column in zip(*parts))
    order = np.lexsort((merchants, customers, kinds, seconds))
    return Schedule(customers[order], merchants[order], seconds[order], kinds[order], groups[order])
