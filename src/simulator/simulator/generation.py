from dataclasses import dataclass
from datetime import date, datetime, time, timedelta, timezone
from uuid import uuid5

import numpy as np

from simulator.arrow import Record
from simulator.config import HistoricalConfig
from simulator.identity import IDENTITY_NAMESPACE, fingerprint
from simulator.profiles import Customer, Profiles, ScheduledPurchase


@dataclass(frozen=True)
class GeneratedHistory:
    events: tuple[Record, ...]
    labels: tuple[Record, ...]
    scenario: str | None


def _scenario_values(
    config: HistoricalConfig,
    purchase: ScheduledPurchase,
    day_offset: int,
    ordinal: int,
    customer: Customer,
    base_amount: int,
    random: np.random.Generator,
) -> tuple[int, str, str, str, str]:
    scenario = purchase.scenario
    amount = base_amount
    device = str(random.choice(customer.familiar_devices))
    network = customer.home_network
    address = customer.shipping_address
    method = customer.payment_method
    if random.random() >= config.behavior.familiar_context_probability:
        device = fingerprint(config.synthetic_key, "device", f"new-{customer.external_id}-{day_offset}-{ordinal}")
        network = fingerprint(config.synthetic_key, "ip_network", f"10.249.{int(random.integers(0, 256))}.1")
        address = fingerprint(config.synthetic_key, "address", f"new-{customer.external_id}-{day_offset}-{ordinal}")
    if not scenario:
        return amount, device, network, address, method
    settings = getattr(config.scenarios, scenario)
    amount = round(amount * settings.amount_multiplier)
    window = purchase.compromise_day if purchase.compromise_day is not None else day_offset
    if scenario == "obvious_high_value":
        amount = max(amount, 750000)
    elif scenario == "stolen_card_new_device":
        device = fingerprint(config.synthetic_key, "device", f"stolen-{customer.external_id}-{window}")
        network = fingerprint(config.synthetic_key, "ip_network", f"10.250.{window % 255}.1")
    elif scenario in ("compromised_device_farm", "card_testing"):
        device = fingerprint(config.synthetic_key, "device", f"shared-farm-{scenario}-{window}")
        network = fingerprint(config.synthetic_key, "ip_network", f"10.251.{window % 255}.1")
        method = fingerprint(config.synthetic_key, "payment_method", f"tok_card-farm-{day_offset}-{ordinal}_ok")
        if scenario == "card_testing":
            amount = int(random.integers(50, 501))
    elif scenario == "account_takeover":
        device = fingerprint(config.synthetic_key, "device", f"takeover-{customer.external_id}-{window}")
        network = fingerprint(config.synthetic_key, "ip_network", f"10.252.{window % 255}.1")
        address = fingerprint(config.synthetic_key, "address", f"takeover-{customer.external_id}-{window}")
    elif scenario == "reshipping_mule":
        address = fingerprint(config.synthetic_key, "address", f"forwarder-{window}")
    return amount, device, network, address, method


def build_history(
    config: HistoricalConfig,
    profiles: Profiles,
    purchase: ScheduledPurchase,
    day: date,
    ordinal: int,
    random: np.random.Generator,
) -> GeneratedHistory:
    customer = profiles.customers[purchase.customer_index]
    merchant = profiles.merchants[purchase.merchant_index]
    payment_id = str(uuid5(IDENTITY_NAMESPACE, f"{config.dataset_version}:{day}:{ordinal}:payment"))
    started_at = datetime.combine(day, time.min, tzinfo=timezone.utc) + timedelta(minutes=purchase.minute)
    history_start = datetime.combine(config.start_date, time.min, tzinfo=timezone.utc)
    day_offset = (day - config.start_date).days
    category_prices = {"electronics": 13000, "home": 7000, "fashion": 4500, "grocery": 2200, "travel": 25000}
    base_amount = max(50, round(float(random.lognormal(np.log(category_prices[merchant.category]), 0.35))))
    if random.random() < config.behavior.small_purchase_probability:
        base_amount = int(random.integers(50, 501))
    amount, device, network, address, method = _scenario_values(
        config, purchase, day_offset, ordinal, customer, base_amount, random
    )
    scenario = purchase.scenario
    fraud = scenario is not None
    events: list[Record] = []
    labels: list[Record] = []
    clock = started_at

    def append(payload: str, at: datetime, **fields) -> None:
        events.append(
            {
                "event_id": str(uuid5(IDENTITY_NAMESPACE, f"{payment_id}:{len(events) + 1}")),
                "payment_id": payment_id,
                "aggregate_version": len(events) + 1,
                "integration_id": str(config.integration_id),
                "schema_version": config.data_problems.schema_evolution.version_on(at.date()),
                "correlation_id": payment_id,
                "causation_id": events[-1]["event_id"] if events else payment_id,
                "occurred_at": at,
                "recorded_at": at,
                payload: fields,
            }
        )

    forwarder = random.random() < 0.1 or scenario == "reshipping_mule"
    append(
        "payment_created",
        clock,
        merchant={"merchant_id": merchant.merchant_id, "external_seller_id": merchant.external_id},
        customer={"customer_id": customer.customer_id, "external_buyer_id": customer.external_id},
        snapshot={
            "external_order_id": f"order-{day.isoformat()}-{ordinal:06d}",
            "total": {"minor_units": amount, "currency": "USD"},
            "merchant_category": merchant.category,
            "shipping": {
                "country": "US",
                "region": "FL" if forwarder else "NY",
                "postal_code": "33166" if forwarder else "10001",
                "address_fingerprint": address,
            },
            "items": [
                {
                    "product_id": f"sku-{merchant.category}-1",
                    "category": merchant.category,
                    "condition": "new",
                    "quantity": 1,
                    "unit_price": {"minor_units": amount, "currency": "USD"},
                }
            ],
            "customer_account_created_at": history_start - timedelta(days=customer.account_age_days),
            "merchant_account_created_at": history_start - timedelta(days=merchant.account_age_days),
        },
    )

    for attempt_index in range(1, 3):
        clock += timedelta(seconds=1)
        attempt_id = str(uuid5(IDENTITY_NAMESPACE, f"{payment_id}:attempt:{attempt_index}"))
        foreign_probability = config.behavior.foreign_ip_probability
        if scenario in ("stolen_card_new_device", "account_takeover"):
            foreign_probability = max(foreign_probability, 0.5)
        signals = {
            "attempted_at": clock,
            "payment_method": {"type": "card", "fingerprint": method},
            "device_fingerprint": device,
            "ip_country": "CA" if random.random() < foreign_probability else "US",
            "ip_network_fingerprint": network,
        }
        if config.data_problems.schema_evolution.version_on(clock.date()) == 2:
            signals["user_agent_family"] = "Chromium"
        append("attempt_started", clock, attempt_id=attempt_id, signals=signals)

        risk_decline = amount >= 500000 and random.random() < 0.5
        clock += timedelta(seconds=1)
        append(
            "risk_decision_recorded",
            clock,
            attempt_id=attempt_id,
            assessment={
                "outcome": "RISK_OUTCOME_DECLINE" if risk_decline else "RISK_OUTCOME_APPROVE",
                "reason_codes": ["HIGH_VALUE" if risk_decline else "ALLOW"],
                "policy_version": "sim-rules-v1",
            },
        )
        clock += timedelta(seconds=1)
        if risk_decline:
            append(
                "attempt_declined",
                clock,
                attempt_id=attempt_id,
                source="DECLINE_SOURCE_RISK",
                reason="high_value",
            )
            succeeded = False
        else:
            append(
                "processor_operation_requested",
                clock,
                attempt_id=attempt_id,
                operation_key=f"{payment_id}:{attempt_index}",
            )
            outcome = random.random()
            unknown = outcome < config.behavior.unknown_outcome_rate
            processor_decline = (
                random.random() < config.behavior.issuer_decline_rate + config.behavior.insufficient_funds_rate
                if unknown
                else outcome
                < config.behavior.unknown_outcome_rate
                + config.behavior.issuer_decline_rate
                + config.behavior.insufficient_funds_rate
            )
            declined = (
                (scenario == "retry_attack" and attempt_index == 1)
                or (scenario == "card_testing" and random.random() < 0.8)
                or processor_decline
            )
            if unknown:
                clock += timedelta(seconds=1)
                append("processor_outcome_unknown", clock, attempt_id=attempt_id)
                clock += timedelta(minutes=config.behavior.unknown_resolution_minutes)
            else:
                clock += timedelta(seconds=1)
            if declined:
                append(
                    "attempt_declined",
                    clock,
                    attempt_id=attempt_id,
                    source="DECLINE_SOURCE_PROCESSOR",
                    reason="issuer_declined",
                )
                succeeded = False
            else:
                append(
                    "payment_succeeded",
                    clock,
                    attempt_id=attempt_id,
                    processor_reference=f"sim-{payment_id}-{attempt_index}",
                )
                succeeded = True

        friendly_fraud = not fraud and succeeded and random.random() < config.scenarios.friendly_fraud.rate
        labels.append(
            {
                "integration_id": str(config.integration_id),
                "payment_id": payment_id,
                "attempt_id": attempt_id,
                "is_fraud": fraud,
                "source": "LABEL_SOURCE_SIMULATION_TRUTH",
                "scenario": "friendly_fraud" if friendly_fraud else scenario or "legitimate",
                "occurred_at": started_at,
                "label_available_at": clock,
            }
        )
        if succeeded and (fraud or friendly_fraud):
            delay = int(random.integers(config.label_delays.min_days, config.label_delays.max_days + 1))
            labels.append(
                {
                    "integration_id": str(config.integration_id),
                    "payment_id": payment_id,
                    "attempt_id": attempt_id,
                    "is_fraud": True,
                    "source": "LABEL_SOURCE_CHARGEBACK",
                    "occurred_at": clock,
                    "label_available_at": clock + timedelta(days=delay),
                }
            )
        if succeeded:
            break
        if attempt_index == 2 or (scenario != "retry_attack" and random.random() >= config.behavior.retry_rate):
            clock = max(clock + timedelta(seconds=1), started_at + timedelta(minutes=config.behavior.expiry_minutes))
            append("payment_expired", clock, _present=True)
            break
        method = fingerprint(config.synthetic_key, "payment_method", f"tok_card-retry-{day_offset}-{ordinal}_ok")

    return GeneratedHistory(tuple(events), tuple(labels), scenario)
