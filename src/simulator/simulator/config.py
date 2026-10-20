from datetime import date, timedelta
from pathlib import Path
from typing import Self
from uuid import UUID

import yaml
from pydantic import BaseModel, ConfigDict, Field, model_validator


class StrictModel(BaseModel):
    model_config = ConfigDict(extra="forbid")


class EntityCounts(StrictModel):
    customers: int = Field(gt=0)
    merchants: int = Field(gt=0)
    devices: int = Field(gt=0)
    ip_networks: int = Field(gt=0, le=65536)
    addresses: int = Field(gt=0)
    payment_methods: int = Field(gt=0)


class PurchaseBehavior(StrictModel):
    purchases_per_customer_day: float = Field(gt=0)
    merchant_zipf_exponent: float = Field(gt=0)
    category_zipf_exponent: float = Field(gt=0)
    familiar_merchant_probability: float = Field(ge=0, le=1)
    familiar_context_probability: float = Field(default=0.9, ge=0, le=1)
    foreign_ip_probability: float = Field(default=0.05, ge=0, le=1)
    small_purchase_probability: float = Field(default=0.03, ge=0, le=1)
    issuer_decline_rate: float = Field(ge=0, le=1)
    insufficient_funds_rate: float = Field(ge=0, le=1)
    unknown_outcome_rate: float = Field(ge=0, le=1)
    unknown_resolution_minutes: int = Field(gt=0)
    retry_rate: float = Field(ge=0, le=1)
    expiry_minutes: int = Field(gt=0)

    @model_validator(mode="after")
    def validate_outcomes(self) -> Self:
        if self.issuer_decline_rate + self.insufficient_funds_rate + self.unknown_outcome_rate > 1:
            raise ValueError("processor outcome rates must sum to at most one")
        return self


class Scenario(StrictModel):
    rate: float = Field(ge=0, le=1)
    window_days: int = Field(default=1, gt=0)
    amount_multiplier: float = Field(default=1, gt=0)


class Scenarios(StrictModel):
    obvious_high_value: Scenario
    stolen_card_new_device: Scenario
    compromised_device_farm: Scenario
    account_takeover: Scenario
    reshipping_mule: Scenario
    card_testing: Scenario
    retry_attack: Scenario
    friendly_fraud: Scenario


class SchemaEvolution(StrictModel):
    cutover_date: date
    field_path: str = "attempt_started.signals.user_agent_family"

    def version_on(self, day: date) -> int:
        return 1 if day < self.cutover_date else 2

    @model_validator(mode="after")
    def validate_field(self) -> Self:
        if self.field_path != "attempt_started.signals.user_agent_family":
            raise ValueError("only attempt_started.signals.user_agent_family is supported for schema evolution")
        return self


class DataProblems(StrictModel):
    duplicate_rate: float = Field(ge=0, lt=1)
    schema_evolution: SchemaEvolution


class LabelDelays(StrictModel):
    min_days: int = Field(ge=0)
    max_days: int = Field(ge=0)

    @model_validator(mode="after")
    def validate_range(self) -> Self:
        if self.min_days > self.max_days:
            raise ValueError("label delay min_days must not exceed max_days")
        return self


class StorageLayout(StrictModel):
    row_group_rows: int = Field(default=65536, gt=0, le=64 * 1024 * 1024)
    target_file_size_mb: int = Field(default=128, gt=0)
    sort_memory_limit_mb: int = Field(default=128, ge=32)


class HistoricalConfig(StrictModel):
    dataset_version: str = Field(pattern=r"^[a-zA-Z0-9][a-zA-Z0-9._-]*$")
    seed: int = Field(ge=0)
    integration_id: UUID
    synthetic_key: str = Field(min_length=1)
    synthetic_key_version: str = Field(min_length=1)
    start_date: date
    duration_days: int = Field(gt=1)
    counts: EntityCounts
    behavior: PurchaseBehavior
    scenarios: Scenarios
    data_problems: DataProblems
    label_delays: LabelDelays
    storage: StorageLayout = Field(default_factory=StorageLayout)
    output: str = Field(min_length=1)

    @model_validator(mode="after")
    def validate_dates(self) -> Self:
        cutover = self.data_problems.schema_evolution.cutover_date
        if not self.start_date < cutover < self.start_date + timedelta(days=self.duration_days):
            raise ValueError("schema_evolution.cutover_date must fall strictly inside the dataset")
        scenario_rates = sum(
            getattr(self.scenarios, name).rate
            for name in self.scenarios.__class__.model_fields
            if name != "friendly_fraud"
        )
        if scenario_rates > 1:
            raise ValueError("fraud scenario rates must sum to at most one")
        return self


def load_config(path: Path) -> HistoricalConfig:
    return HistoricalConfig.model_validate(yaml.safe_load(path.read_text(encoding="utf-8")))
