from __future__ import annotations

from datetime import timedelta

from app.domain.decision import DecisionInput, DecisionResult


def evaluate(decision_input: DecisionInput) -> DecisionResult:
    reasons = []
    if decision_input.amount_minor > 200_000:
        reasons.append("HIGH_AMOUNT")
    if (
        decision_input.ip_country is not None
        and decision_input.ip_country != decision_input.shipping_country
        and decision_input.attempted_at - decision_input.customer_account_created_at < timedelta(days=7)
        and decision_input.amount_minor > 20_000
    ):
        reasons.append("NEW_ACCOUNT_GEO_MISMATCH")
    return DecisionResult(outcome="DECLINE" if reasons else "APPROVE", reason_codes=tuple(reasons))
