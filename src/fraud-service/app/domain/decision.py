from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class DecisionInput(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    integration_id: str = Field(min_length=1)
    payment_id: str = Field(min_length=1)
    attempt_id: str = Field(min_length=1)
    merchant_id: str = Field(min_length=1)
    external_seller_id: str = Field(min_length=1)
    customer_id: str = Field(min_length=1)
    external_buyer_id: str = Field(min_length=1)
    external_order_id: str = Field(min_length=1)
    merchant_category: str = Field(min_length=1)
    merchant_account_created_at: datetime
    customer_account_created_at: datetime
    attempted_at: datetime
    amount_minor: int = Field(ge=0)
    currency: Literal["USD"]
    shipping_country: str = Field(min_length=1)
    shipping_region: str = Field(min_length=1)
    shipping_postal_code: str = Field(min_length=1)
    address_fingerprint: str = Field(min_length=1)
    payment_method_type: str = Field(min_length=1)
    payment_method_fingerprint: str = Field(min_length=1)
    ip_country: str | None = None

    @field_validator("ip_country")
    @classmethod
    def validate_ip_country(cls, country: str | None) -> str | None:
        if country == "":
            raise ValueError("IP country must be absent when unknown")
        return country

    @model_validator(mode="after")
    def validate_times(self) -> DecisionInput:
        if self.customer_account_created_at > self.attempted_at:
            raise ValueError("account was created after attempt")
        if self.merchant_account_created_at > self.attempted_at:
            raise ValueError("merchant was created after attempt")
        return self


class DecisionResult(BaseModel):
    model_config = ConfigDict(frozen=True)

    outcome: Literal["APPROVE", "DECLINE"]
    reason_codes: tuple[str, ...]
    policy_version: Literal["rules-v1"] = "rules-v1"
