from __future__ import annotations

from datetime import UTC

from app.domain.decision import DecisionInput
from fraud.v2 import fraud_pb2


def decision_input_from_proto(request: fraud_pb2.DecideRequest) -> DecisionInput:
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
        or sum(item.quantity * item.unit_price.minor_units for item in snapshot.items) != snapshot.total.minor_units
    ):
        raise ValueError("invalid checkout items")
    return DecisionInput.model_validate(
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
