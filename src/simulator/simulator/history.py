from collections.abc import Sequence

from payments.v1.events_pb2 import (
    DECLINE_SOURCE_PROCESSOR,
    DECLINE_SOURCE_RISK,
    RISK_OUTCOME_APPROVE,
    RISK_OUTCOME_DECLINE,
    PaymentEvent,
)


class HistoryValidationError(ValueError):
    def __init__(self, payment_id: str, rule: str):
        super().__init__(f"payment {payment_id}: {rule}")
        self.rule = rule


def validate_history(events: Sequence[PaymentEvent]) -> None:
    if not events:
        raise HistoryValidationError("<empty>", "missing_created")

    payment_id = events[0].payment_id
    state = "initial"
    attempt_id: str | None = None
    seen_attempt_ids: set[str] = set()
    seen_event_ids: set[str] = set()
    last_decline_source = None

    for version, event in enumerate(events, start=1):
        payload = event.WhichOneof("payload")
        if event.payment_id != payment_id:
            raise HistoryValidationError(payment_id, "mixed_payment_ids")
        if event.event_id in seen_event_ids:
            raise HistoryValidationError(payment_id, "duplicate_event_id")
        seen_event_ids.add(event.event_id)
        if event.aggregate_version != version:
            raise HistoryValidationError(payment_id, "version_gap")
        if state == "initial":
            if payload != "payment_created":
                raise HistoryValidationError(payment_id, "missing_created")
            state = "open"
            continue
        if state == "terminal":
            raise HistoryValidationError(payment_id, "after_terminal")
        if payload == "payment_created":
            raise HistoryValidationError(payment_id, "second_created")
        if payload == "payment_expired":
            if state in ("processor_pending", "unknown"):
                raise HistoryValidationError(payment_id, "expiry_during_unknown")
            state = "terminal"
            continue

        if payload == "attempt_started":
            if state != "open":
                rule = "retry_while_unknown" if state == "unknown" else "concurrent_attempt"
                raise HistoryValidationError(payment_id, rule)
            if not event.attempt_started.attempt_id or event.attempt_started.attempt_id in seen_attempt_ids:
                raise HistoryValidationError(payment_id, "reused_attempt_id")
            attempt_id = event.attempt_started.attempt_id
            seen_attempt_ids.add(attempt_id)
            state = "risk_pending"
            continue

        if payload == "risk_decision_recorded":
            if state != "risk_pending":
                raise HistoryValidationError(payment_id, "decision_without_attempt")
            assessment = event.risk_decision_recorded.assessment
            if event.risk_decision_recorded.attempt_id != attempt_id:
                raise HistoryValidationError(payment_id, "wrong_attempt")
            if not assessment.policy_version or not assessment.reason_codes:
                raise HistoryValidationError(payment_id, "incomplete_risk_assessment")
            if assessment.outcome == RISK_OUTCOME_APPROVE:
                state = "approved"
            elif assessment.outcome == RISK_OUTCOME_DECLINE:
                state = "risk_declined"
            else:
                raise HistoryValidationError(payment_id, "unspecified_risk_outcome")
            continue

        if payload == "processor_operation_requested":
            if state != "approved":
                rule = (
                    "processor_after_risk_decline"
                    if last_decline_source == DECLINE_SOURCE_RISK
                    else "processor_without_approval"
                )
                raise HistoryValidationError(payment_id, rule)
            if event.processor_operation_requested.attempt_id != attempt_id:
                raise HistoryValidationError(payment_id, "wrong_attempt")
            state = "processor_pending"
            continue

        if payload == "processor_outcome_unknown":
            if state != "processor_pending":
                raise HistoryValidationError(payment_id, "unknown_without_request")
            if event.processor_outcome_unknown.attempt_id != attempt_id:
                raise HistoryValidationError(payment_id, "wrong_attempt")
            state = "unknown"
            continue

        if payload == "attempt_declined":
            decline = event.attempt_declined
            if decline.attempt_id != attempt_id:
                raise HistoryValidationError(payment_id, "wrong_attempt")
            expected_source = DECLINE_SOURCE_RISK if state == "risk_declined" else DECLINE_SOURCE_PROCESSOR
            if state not in ("risk_declined", "processor_pending", "unknown") or decline.source != expected_source:
                raise HistoryValidationError(payment_id, "wrong_decline_source")
            last_decline_source = decline.source
            state = "open"
            continue

        if payload == "payment_succeeded":
            if state not in ("processor_pending", "unknown"):
                raise HistoryValidationError(payment_id, "success_without_processor_request")
            if event.payment_succeeded.attempt_id != attempt_id:
                raise HistoryValidationError(payment_id, "wrong_attempt")
            state = "terminal"
            continue

        raise HistoryValidationError(payment_id, "unexpected_transition")

    if state != "terminal":
        raise HistoryValidationError(payment_id, "incomplete_history")
