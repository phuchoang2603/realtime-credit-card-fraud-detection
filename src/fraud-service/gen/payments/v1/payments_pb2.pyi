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

class PaymentStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PAYMENT_STATUS_UNSPECIFIED: _ClassVar[PaymentStatus]
    PAYMENT_STATUS_OPEN: _ClassVar[PaymentStatus]
    PAYMENT_STATUS_SUCCEEDED: _ClassVar[PaymentStatus]
    PAYMENT_STATUS_EXPIRED: _ClassVar[PaymentStatus]

class AttemptStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ATTEMPT_STATUS_UNSPECIFIED: _ClassVar[AttemptStatus]
    ATTEMPT_STATUS_PENDING: _ClassVar[AttemptStatus]
    ATTEMPT_STATUS_SUCCEEDED: _ClassVar[AttemptStatus]
    ATTEMPT_STATUS_DECLINED: _ClassVar[AttemptStatus]
PAYMENT_STATUS_UNSPECIFIED: PaymentStatus
PAYMENT_STATUS_OPEN: PaymentStatus
PAYMENT_STATUS_SUCCEEDED: PaymentStatus
PAYMENT_STATUS_EXPIRED: PaymentStatus
ATTEMPT_STATUS_UNSPECIFIED: AttemptStatus
ATTEMPT_STATUS_PENDING: AttemptStatus
ATTEMPT_STATUS_SUCCEEDED: AttemptStatus
ATTEMPT_STATUS_DECLINED: AttemptStatus

class CreatePaymentRequest(_message.Message):
    __slots__ = ("integration_id", "idempotency_key", "external_seller_id", "external_buyer_id", "snapshot", "return_url")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_SELLER_ID_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_BUYER_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    RETURN_URL_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    idempotency_key: str
    external_seller_id: str
    external_buyer_id: str
    snapshot: _types_pb2.CheckoutSnapshot
    return_url: str
    def __init__(self, integration_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., external_seller_id: _Optional[str] = ..., external_buyer_id: _Optional[str] = ..., snapshot: _Optional[_Union[_types_pb2.CheckoutSnapshot, _Mapping]] = ..., return_url: _Optional[str] = ...) -> None: ...

class StartAttemptRequest(_message.Message):
    __slots__ = ("integration_id", "session_id", "idempotency_key", "payment_method_token", "signals")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PAYMENT_METHOD_TOKEN_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    session_id: str
    idempotency_key: str
    payment_method_token: str
    signals: _types_pb2.AttemptSignals
    def __init__(self, integration_id: _Optional[str] = ..., session_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., payment_method_token: _Optional[str] = ..., signals: _Optional[_Union[_types_pb2.AttemptSignals, _Mapping]] = ...) -> None: ...

class GetPaymentRequest(_message.Message):
    __slots__ = ("integration_id", "payment_id")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    PAYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    payment_id: str
    def __init__(self, integration_id: _Optional[str] = ..., payment_id: _Optional[str] = ...) -> None: ...

class AttemptResult(_message.Message):
    __slots__ = ("attempt_id", "status", "decline_reason")
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DECLINE_REASON_FIELD_NUMBER: _ClassVar[int]
    attempt_id: str
    status: AttemptStatus
    decline_reason: str
    def __init__(self, attempt_id: _Optional[str] = ..., status: _Optional[_Union[AttemptStatus, str]] = ..., decline_reason: _Optional[str] = ...) -> None: ...

class Payment(_message.Message):
    __slots__ = ("payment_id", "session_id", "checkout_token", "expires_at", "status", "version", "merchant", "customer", "snapshot", "return_url", "attempts")
    PAYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHECKOUT_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MERCHANT_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    RETURN_URL_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    payment_id: str
    session_id: str
    checkout_token: str
    expires_at: _timestamp_pb2.Timestamp
    status: PaymentStatus
    version: int
    merchant: _types_pb2.MerchantRef
    customer: _types_pb2.CustomerRef
    snapshot: _types_pb2.CheckoutSnapshot
    return_url: str
    attempts: _containers.RepeatedCompositeFieldContainer[AttemptResult]
    def __init__(self, payment_id: _Optional[str] = ..., session_id: _Optional[str] = ..., checkout_token: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[PaymentStatus, str]] = ..., version: _Optional[int] = ..., merchant: _Optional[_Union[_types_pb2.MerchantRef, _Mapping]] = ..., customer: _Optional[_Union[_types_pb2.CustomerRef, _Mapping]] = ..., snapshot: _Optional[_Union[_types_pb2.CheckoutSnapshot, _Mapping]] = ..., return_url: _Optional[str] = ..., attempts: _Optional[_Iterable[_Union[AttemptResult, _Mapping]]] = ...) -> None: ...
