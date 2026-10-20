import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from payments.v1 import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DecisionOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DECISION_OUTCOME_UNSPECIFIED: _ClassVar[DecisionOutcome]
    DECISION_OUTCOME_APPROVE: _ClassVar[DecisionOutcome]
    DECISION_OUTCOME_DECLINE: _ClassVar[DecisionOutcome]
DECISION_OUTCOME_UNSPECIFIED: DecisionOutcome
DECISION_OUTCOME_APPROVE: DecisionOutcome
DECISION_OUTCOME_DECLINE: DecisionOutcome

class DecideRequest(_message.Message):
    __slots__ = ("integration_id", "payment_id", "attempt_id", "merchant", "customer", "snapshot", "signals")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    PAYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    MERCHANT_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    payment_id: str
    attempt_id: str
    merchant: _types_pb2.MerchantRef
    customer: _types_pb2.CustomerRef
    snapshot: _types_pb2.CheckoutSnapshot
    signals: _types_pb2.AttemptSignals
    def __init__(self, integration_id: _Optional[str] = ..., payment_id: _Optional[str] = ..., attempt_id: _Optional[str] = ..., merchant: _Optional[_Union[_types_pb2.MerchantRef, _Mapping]] = ..., customer: _Optional[_Union[_types_pb2.CustomerRef, _Mapping]] = ..., snapshot: _Optional[_Union[_types_pb2.CheckoutSnapshot, _Mapping]] = ..., signals: _Optional[_Union[_types_pb2.AttemptSignals, _Mapping]] = ...) -> None: ...

class DecideResponse(_message.Message):
    __slots__ = ("outcome", "reason_codes", "evaluated_at", "policy_version", "model_version", "feature_set_version", "risk_score")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REASON_CODES_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    FEATURE_SET_VERSION_FIELD_NUMBER: _ClassVar[int]
    RISK_SCORE_FIELD_NUMBER: _ClassVar[int]
    outcome: DecisionOutcome
    reason_codes: _containers.RepeatedScalarFieldContainer[str]
    evaluated_at: _timestamp_pb2.Timestamp
    policy_version: str
    model_version: str
    feature_set_version: str
    risk_score: float
    def __init__(self, outcome: _Optional[_Union[DecisionOutcome, str]] = ..., reason_codes: _Optional[_Iterable[str]] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., policy_version: _Optional[str] = ..., model_version: _Optional[str] = ..., feature_set_version: _Optional[str] = ..., risk_score: _Optional[float] = ...) -> None: ...
