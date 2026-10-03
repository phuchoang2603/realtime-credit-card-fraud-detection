import pytest
from pydantic import ValidationError

from app.domain.policy import evaluate
from app.transport.mapping import decision_input_from_proto


@pytest.mark.parametrize(
    ("amount", "expected"),
    [(200_000, "APPROVE"), (200_001, "DECLINE")],
)
def test_high_amount_boundary(valid_request, amount, expected):
    valid_request.snapshot.total.minor_units = amount
    valid_request.snapshot.items[0].unit_price.minor_units = amount
    result = evaluate(decision_input_from_proto(valid_request))
    assert result.outcome == expected
    assert result.reason_codes == (() if expected == "APPROVE" else ("HIGH_AMOUNT",))


@pytest.mark.parametrize("field", ["integration_id", "payment_id", "attempt_id"])
def test_missing_identifier_is_rejected(valid_request, field):
    setattr(valid_request, field, "")
    with pytest.raises(ValidationError):
        decision_input_from_proto(valid_request)


def test_missing_amount_is_rejected(valid_request):
    valid_request.snapshot.ClearField("total")
    with pytest.raises(ValueError):
        decision_input_from_proto(valid_request)


def test_unsupported_currency_is_rejected(valid_request):
    valid_request.snapshot.total.currency = "EUR"
    valid_request.snapshot.items[0].unit_price.currency = "EUR"
    with pytest.raises(ValidationError):
        decision_input_from_proto(valid_request)


def test_snapshot_sum_is_validated(valid_request):
    valid_request.snapshot.total.minor_units += 1
    with pytest.raises(ValueError):
        decision_input_from_proto(valid_request)
