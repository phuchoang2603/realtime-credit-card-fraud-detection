import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from payments.v1 import types_pb2 as _types_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
    __slots__ = ("assessment", "evaluated_at")
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    assessment: _types_pb2.RiskAssessment
    evaluated_at: _timestamp_pb2.Timestamp
    def __init__(self, assessment: _Optional[_Union[_types_pb2.RiskAssessment, _Mapping]] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
