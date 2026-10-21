import json
from dataclasses import asdict, dataclass, replace
from datetime import date
from hashlib import sha256

import numpy as np

from simulator.config import HistoricalConfig
from simulator.identity import entity_id, fingerprint

CATEGORIES = ("electronics", "home", "fashion", "grocery", "travel")
SCENARIO_NAMES = (
    "obvious_high_value",
    "stolen_card_new_device",
    "compromised_device_farm",
    "account_takeover",
    "reshipping_mule",
    "card_testing",
    "retry_attack",
)


def stage_rng(seed: int, stage: int, day: int | None = None) -> np.random.Generator:
    return np.random.default_rng(np.random.SeedSequence([seed, stage, *([] if day is None else [day])]))


def zipf_weights(count: int, exponent: float) -> np.ndarray:
    weights = np.arange(1, count + 1, dtype=float) ** (-exponent)
    return weights / weights.sum()


@dataclass(frozen=True)
class Merchant:
    external_id: str
    merchant_id: str
    category: str
    account_age_days: int


@dataclass(frozen=True)
class Customer:
    external_id: str
    customer_id: str
    account_age_days: int
    familiar_merchants: tuple[int, ...]
    familiar_devices: tuple[str, ...]
    home_network: str
    shipping_address: str
    payment_method: str


@dataclass(frozen=True)
class Profiles:
    merchants: tuple[Merchant, ...]
    customers: tuple[Customer, ...]

    def digest(self) -> str:
        canonical = json.dumps(asdict(self), sort_keys=True, separators=(",", ":"))
        return sha256(canonical.encode()).hexdigest()


@dataclass(frozen=True)
class ScheduledPurchase:
    customer_index: int
    merchant_index: int
    minute: int
    scenario: str | None
    compromise_day: int | None = None


def build_profiles(config: HistoricalConfig) -> Profiles:
    merchant_rng = stage_rng(config.seed, 1)
    customer_rng = stage_rng(config.seed, 2)
    category_weights = zipf_weights(len(CATEGORIES), config.behavior.category_zipf_exponent)
    merchant_weights = zipf_weights(config.counts.merchants, config.behavior.merchant_zipf_exponent)
    merchants = tuple(
        Merchant(
            external_id=f"seller-{index + 1:06d}",
            merchant_id=entity_id(config.integration_id, "merchant", f"seller-{index + 1:06d}"),
            category=str(merchant_rng.choice(CATEGORIES, p=category_weights)),
            account_age_days=int(merchant_rng.integers(30, 1500)),
        )
        for index in range(config.counts.merchants)
    )
    device_pool = tuple(
        fingerprint(config.synthetic_key, "device", f"device-{index:06d}") for index in range(config.counts.devices)
    )
    network_pool = tuple(
        fingerprint(config.synthetic_key, "ip_network", f"10.{index // 256}.{index % 256}.1")
        for index in range(config.counts.ip_networks)
    )
    address_pool = tuple(
        fingerprint(config.synthetic_key, "address", f"{index} Market Street, Example City")
        for index in range(config.counts.addresses)
    )
    method_pool = tuple(
        fingerprint(config.synthetic_key, "payment_method", f"tok_card-{index:06d}_ok")
        for index in range(config.counts.payment_methods)
    )
    customers = tuple(
        Customer(
            external_id=f"buyer-{index + 1:06d}",
            customer_id=entity_id(config.integration_id, "customer", f"buyer-{index + 1:06d}"),
            account_age_days=int(customer_rng.integers(1, 2500)),
            familiar_merchants=tuple(
                sorted(
                    int(value)
                    for value in customer_rng.choice(
                        config.counts.merchants,
                        size=min(3, config.counts.merchants),
                        replace=False,
                        p=merchant_weights,
                    )
                )
            ),
            familiar_devices=tuple(
                device_pool[int(value)]
                for value in customer_rng.choice(
                    config.counts.devices, size=min(2, config.counts.devices), replace=False
                )
            ),
            home_network=network_pool[index % len(network_pool)],
            shipping_address=address_pool[index % len(address_pool)],
            payment_method=method_pool[index % len(method_pool)],
        )
        for index in range(config.counts.customers)
    )
    return Profiles(merchants=merchants, customers=customers)


def compromises_for_day(config: HistoricalConfig, offset: int) -> dict[int, tuple[str, int]]:
    compromised = {}
    for stage, name in enumerate(SCENARIO_NAMES, start=6):
        settings = getattr(config.scenarios, name)
        if not settings.rate:
            continue
        activation_rate = 1 - (1 - settings.rate) ** (1 / settings.window_days)
        for start in range(offset - settings.window_days + 1, offset + 1):
            random = stage_rng(config.seed, stage, start + settings.window_days)
            for customer in np.flatnonzero(random.random(config.counts.customers) < activation_rate):
                index = int(customer)
                if index not in compromised or compromised[index][0] == name:
                    compromised[index] = (name, start)
    return compromised


def schedule_for_day(config: HistoricalConfig, day: date, profiles: Profiles) -> tuple[ScheduledPurchase, ...]:
    offset = (day - config.start_date).days
    if not 0 <= offset < config.duration_days:
        raise ValueError("day is outside the configured dataset")
    random = stage_rng(config.seed, 3, offset)
    merchant_weights = zipf_weights(len(profiles.merchants), config.behavior.merchant_zipf_exponent)
    compromised = compromises_for_day(config, offset)
    frequency = config.behavior.purchases_per_customer_day * (config.drift.factor_on(day) if config.drift else 1.0)
    scheduled = []
    for customer_index, customer in enumerate(profiles.customers):
        for _ in range(int(random.poisson(frequency))):
            if random.random() < config.behavior.familiar_merchant_probability:
                merchant_index = int(random.choice(customer.familiar_merchants))
            else:
                merchant_index = int(random.choice(len(profiles.merchants), p=merchant_weights))
            scenario, compromise_day = compromised.get(customer_index, (None, None))
            scheduled.append(
                ScheduledPurchase(
                    customer_index, merchant_index, int(random.integers(0, 1440)), scenario, compromise_day
                )
            )
    if any(purchase.scenario == "card_testing" for purchase in scheduled):
        burst_minute = int(random.integers(0, 1430))
        burst_index = 0
        for index, purchase in enumerate(scheduled):
            if purchase.scenario == "card_testing":
                scheduled[index] = replace(purchase, minute=burst_minute + burst_index % 10)
                burst_index += 1
    return tuple(
        sorted(scheduled, key=lambda purchase: (purchase.minute, purchase.customer_index, purchase.merchant_index))
    )
