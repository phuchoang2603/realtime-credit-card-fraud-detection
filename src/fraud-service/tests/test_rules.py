from datetime import timedelta

import pytest

from app.domain.policy import evaluate
from app.transport.mapping import decision_input_from_proto


@pytest.mark.parametrize(
    ("amount", "age_days", "matched"),
    [(20_000, 1, False), (20_001, 1, True), (20_001, 7, False)],
)
def test_geo_mismatch_boundaries(valid_request, amount, age_days, matched):
    valid_request.snapshot.total.minor_units = amount
    valid_request.snapshot.items[0].unit_price.minor_units = amount
    valid_request.snapshot.customer_account_created_at.FromDatetime(
        valid_request.signals.attempted_at.ToDatetime() - timedelta(days=age_days)
    )
    valid_request.signals.ip_country = "CA"
    result = evaluate(decision_input_from_proto(valid_request))
    assert ("NEW_ACCOUNT_GEO_MISMATCH" in result.reason_codes) is matched


def test_unknown_ip_country_does_not_match_geo_rule(valid_request):
    valid_request.snapshot.customer_account_created_at.CopyFrom(valid_request.signals.attempted_at)
    result = evaluate(decision_input_from_proto(valid_request))
    assert result.outcome == "APPROVE"


def test_all_matching_reasons_and_no_model_fields(valid_request):
    valid_request.snapshot.total.minor_units = 200_001
    valid_request.snapshot.items[0].unit_price.minor_units = 200_001
    valid_request.snapshot.customer_account_created_at.CopyFrom(valid_request.signals.attempted_at)
    valid_request.signals.ip_country = "CA"
    result = evaluate(decision_input_from_proto(valid_request))
    assert result.reason_codes == ("HIGH_AMOUNT", "NEW_ACCOUNT_GEO_MISMATCH")
    assert result.policy_version == "rules-v1"
