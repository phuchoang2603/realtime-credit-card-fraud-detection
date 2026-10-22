import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATION_OUTCOME_UNSPECIFIED: _ClassVar[OperationOutcome]
    OPERATION_OUTCOME_APPROVED: _ClassVar[OperationOutcome]
    OPERATION_OUTCOME_DECLINED: _ClassVar[OperationOutcome]
OPERATION_OUTCOME_UNSPECIFIED: OperationOutcome
OPERATION_OUTCOME_APPROVED: OperationOutcome
OPERATION_OUTCOME_DECLINED: OperationOutcome

class SaleRequest(_message.Message):
    __slots__ = ("operation_key", "integration_id", "payment_method_token", "amount_minor_units", "currency")
    OPERATION_KEY_FIELD_NUMBER: _ClassVar[int]
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    PAYMENT_METHOD_TOKEN_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_UNITS_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    operation_key: str
    integration_id: str
    payment_method_token: str
    amount_minor_units: int
    currency: str
    def __init__(self, operation_key: _Optional[str] = ..., integration_id: _Optional[str] = ..., payment_method_token: _Optional[str] = ..., amount_minor_units: _Optional[int] = ..., currency: _Optional[str] = ...) -> None: ...

class GetOperationRequest(_message.Message):
    __slots__ = ("operation_key",)
    OPERATION_KEY_FIELD_NUMBER: _ClassVar[int]
    operation_key: str
    def __init__(self, operation_key: _Optional[str] = ...) -> None: ...

class Operation(_message.Message):
    __slots__ = ("operation_key", "outcome", "decline_reason", "processor_reference", "recorded_at")
    OPERATION_KEY_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DECLINE_REASON_FIELD_NUMBER: _ClassVar[int]
    PROCESSOR_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    operation_key: str
    outcome: OperationOutcome
    decline_reason: str
    processor_reference: str
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, operation_key: _Optional[str] = ..., outcome: _Optional[_Union[OperationOutcome, str]] = ..., decline_reason: _Optional[str] = ..., processor_reference: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
