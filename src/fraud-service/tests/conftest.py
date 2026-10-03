from datetime import UTC, datetime, timedelta

import pytest
from google.protobuf.timestamp_pb2 import Timestamp

from fraud.v2 import fraud_pb2
from payments.v1 import types_pb2


def timestamp(value: datetime) -> Timestamp:
    result = Timestamp()
    result.FromDatetime(value)
    return result


@pytest.fixture
def valid_request():
    attempted_at = datetime(2026, 9, 1, tzinfo=UTC)
    return fraud_pb2.DecideRequest(
        integration_id="synthetic-v1",
        payment_id="payment-1",
        attempt_id="attempt-1",
        merchant=types_pb2.MerchantRef(merchant_id="merchant-1", external_seller_id="seller-1"),
        customer=types_pb2.CustomerRef(customer_id="customer-1", external_buyer_id="buyer-1"),
        snapshot=types_pb2.CheckoutSnapshot(
            external_order_id="order-1",
            total=types_pb2.Money(minor_units=15_000, currency="USD"),
            items=[
                types_pb2.LineItem(
                    product_id="product-1",
                    category="electronics",
                    condition="used",
                    quantity=1,
                    unit_price=types_pb2.Money(minor_units=15_000, currency="USD"),
                )
            ],
            merchant_category="electronics",
            merchant_account_created_at=timestamp(attempted_at - timedelta(days=30)),
            customer_account_created_at=timestamp(attempted_at - timedelta(days=30)),
            shipping=types_pb2.ShippingLocation(
                country="US", region="NY", postal_code="10001", address_fingerprint="address-hash"
            ),
        ),
        signals=types_pb2.AttemptSignals(
            attempted_at=timestamp(attempted_at),
            payment_method=types_pb2.PaymentMethod(type="card", fingerprint="method-hash"),
        ),
    )
