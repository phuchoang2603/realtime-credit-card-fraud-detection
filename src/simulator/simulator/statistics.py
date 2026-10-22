from collections import Counter
from dataclasses import dataclass, field
from datetime import datetime

from simulator.generation import GeneratedHistory

IDENTITIES = ("customer", "device", "ip_network", "address", "payment_method")


@dataclass
class DatasetStatistics:
    payments: int = 0
    unique_events: int = 0
    duplicate_events: int = 0
    labels: int = 0
    chargebacks: int = 0
    history_end: datetime | None = None
    field_present_before_cutover: int = 0
    field_present_after_cutover: int = 0
    scenarios: Counter = field(default_factory=Counter)
    merchants: Counter = field(default_factory=Counter)
    categories: Counter = field(default_factory=Counter)
    identities: dict[str, set[str]] = field(default_factory=lambda: {name: set() for name in IDENTITIES})

    def observe(self, history: GeneratedHistory, duplicates: int) -> None:
        self.payments += 1
        self.duplicate_events += duplicates
        self.scenarios[history.truth] += 1
        self.labels += len(history.labels)
        self.chargebacks += sum(label["source"] == "LABEL_SOURCE_CHARGEBACK" for label in history.labels)
        for event in history.events:
            self.unique_events += 1
            occurred = event["occurred_at"]
            self.history_end = occurred if self.history_end is None else max(self.history_end, occurred)
            if (created := event.get("payment_created")) is not None:
                self.merchants[created["merchant"]["merchant_id"]] += 1
                self.categories[created["snapshot"]["items"][0]["category"]] += 1
                self.identities["customer"].add(created["customer"]["customer_id"])
                self.identities["address"].add(created["snapshot"]["shipping"]["address_fingerprint"])
            elif (started := event.get("attempt_started")) is not None:
                signals = started["signals"]
                self.identities["device"].add(signals["device_fingerprint"])
                self.identities["ip_network"].add(signals["ip_network_fingerprint"])
                self.identities["payment_method"].add(signals["payment_method"]["fingerprint"])
                present = signals.get("user_agent_family") is not None
                if event["schema_version"] == 1:
                    self.field_present_before_cutover += present
                else:
                    self.field_present_after_cutover += present

    def observed(self) -> dict:
        exported = self.unique_events + self.duplicate_events
        return {
            "payments": self.payments,
            "unique_events": self.unique_events,
            "duplicate_events": self.duplicate_events,
            "duplicate_rate": self.duplicate_events / exported if exported else 0,
            "labels": self.labels,
            "chargebacks": self.chargebacks,
            "scenario_payments": dict(sorted(self.scenarios.items())),
            "category_payments": dict(sorted(self.categories.items())),
            "merchant_count": len(self.merchants),
            "merchant_max_share": max(self.merchants.values()) / self.payments if self.payments else 0,
            "distinct": {name: len(values) for name, values in self.identities.items()},
            "user_agent_family_present_before_cutover": self.field_present_before_cutover,
            "user_agent_family_present_after_cutover": self.field_present_after_cutover,
        }
