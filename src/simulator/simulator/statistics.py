from collections import Counter
from dataclasses import dataclass, field
from datetime import datetime

from simulator.generation import GeneratedHistory


@dataclass
class DatasetStatistics:
    unique_events: int = 0
    history_end: datetime | None = None
    chargebacks: int = 0
    schema_field_present_before: int = 0
    schema_field_present_after: int = 0
    scenarios: Counter = field(default_factory=Counter)
    merchants: Counter = field(default_factory=Counter)
    categories: Counter = field(default_factory=Counter)
    daily_purchases: Counter = field(default_factory=Counter)
    identities: dict[str, set[str]] = field(
        default_factory=lambda: {
            name: set() for name in ("customer", "device", "ip_network", "address", "payment_method")
        }
    )

    def observe(self, history: GeneratedHistory) -> None:
        friendly_fraud = any(label.get("scenario") == "friendly_fraud" for label in history.labels)
        self.scenarios["friendly_fraud" if friendly_fraud else history.scenario or "legitimate"] += 1
        self.chargebacks += sum(label["source"] == "LABEL_SOURCE_CHARGEBACK" for label in history.labels)
        for event in history.events:
            self.unique_events += 1
            timestamp = event["occurred_at"]
            self.history_end = max(self.history_end, timestamp) if self.history_end is not None else timestamp
            if event.get("payment_created") is not None:
                created = event["payment_created"]
                self.daily_purchases[timestamp.date().isoformat()] += 1
                self.merchants[created["merchant"]["merchant_id"]] += 1
                self.categories[created["snapshot"]["merchant_category"]] += 1
                self.identities["customer"].add(created["customer"]["customer_id"])
                self.identities["address"].add(created["snapshot"]["shipping"]["address_fingerprint"])
            elif event.get("attempt_started") is not None:
                signals = event["attempt_started"]["signals"]
                self.identities["device"].add(signals["device_fingerprint"])
                self.identities["ip_network"].add(signals["ip_network_fingerprint"])
                self.identities["payment_method"].add(signals["payment_method"]["fingerprint"])
                if event["schema_version"] == 1:
                    self.schema_field_present_before += signals.get("user_agent_family") is not None
                else:
                    self.schema_field_present_after += signals.get("user_agent_family") is not None

    def observed(self, duplicate_events: int) -> dict:
        total = self.unique_events + duplicate_events
        purchases = self.merchants.total()
        return {
            "duplicate_rate": duplicate_events / total if total else 0,
            "duplicate_events": duplicate_events,
            "scenario_counts": dict(sorted(self.scenarios.items())),
            "daily_purchase_counts": dict(sorted(self.daily_purchases.items())),
            "merchant_count": len(self.merchants),
            **{f"{name}_count": len(values) for name, values in self.identities.items()},
            "category_counts": dict(sorted(self.categories.items())),
            "merchant_max_share": max(self.merchants.values()) / purchases if purchases else 0,
            "schema_field_present_before": self.schema_field_present_before,
            "schema_field_present_after": self.schema_field_present_after,
            "chargebacks": self.chargebacks,
        }
