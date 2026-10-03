from __future__ import annotations

from datetime import UTC, datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from fraud.v2 import fraud_pb2


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

    @classmethod
    def from_proto(cls, request: fraud_pb2.DecideRequest) -> DecisionInput:
        if not all(request.HasField(field) for field in ("merchant", "customer", "snapshot", "signals")):
            raise ValueError("missing decision context")
        snapshot = request.snapshot
        signals = request.signals
        if not all(
            snapshot.HasField(field)
            for field in ("total", "merchant_account_created_at", "customer_account_created_at", "shipping")
        ) or not all(signals.HasField(field) for field in ("attempted_at", "payment_method")):
            raise ValueError("missing required decision field")
        if (
            not snapshot.items
            or any(
                not item.HasField("unit_price")
                or item.quantity <= 0
                or not item.product_id
                or not item.category
                or not item.condition
                or item.unit_price.minor_units < 0
                or item.unit_price.currency != snapshot.total.currency
                for item in snapshot.items
            )
            or sum(item.quantity * item.unit_price.minor_units for item in snapshot.items)
            != snapshot.total.minor_units
        ):
            raise ValueError("invalid checkout items")
        return cls.model_validate(
            {
                "integration_id": request.integration_id,
                "payment_id": request.payment_id,
                "attempt_id": request.attempt_id,
                "merchant_id": request.merchant.merchant_id,
                "external_seller_id": request.merchant.external_seller_id,
                "customer_id": request.customer.customer_id,
                "external_buyer_id": request.customer.external_buyer_id,
                "external_order_id": snapshot.external_order_id,
                "merchant_category": snapshot.merchant_category,
                "merchant_account_created_at": snapshot.merchant_account_created_at.ToDatetime(tzinfo=UTC),
                "customer_account_created_at": snapshot.customer_account_created_at.ToDatetime(tzinfo=UTC),
                "attempted_at": signals.attempted_at.ToDatetime(tzinfo=UTC),
                "amount_minor": snapshot.total.minor_units,
                "currency": snapshot.total.currency,
                "shipping_country": snapshot.shipping.country,
                "shipping_region": snapshot.shipping.region,
                "shipping_postal_code": snapshot.shipping.postal_code,
                "address_fingerprint": snapshot.shipping.address_fingerprint,
                "payment_method_type": signals.payment_method.type,
                "payment_method_fingerprint": signals.payment_method.fingerprint,
                "ip_country": signals.ip_country if signals.HasField("ip_country") else None,
            }
        )


class DecisionResult(BaseModel):
    model_config = ConfigDict(frozen=True)

    outcome: Literal["APPROVE", "DECLINE"]
    reason_codes: tuple[str, ...]
    policy_version: Literal["rules-v1"] = "rules-v1"
