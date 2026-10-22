from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Behavior(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BEHAVIOR_UNSPECIFIED: _ClassVar[Behavior]
    BEHAVIOR_APPROVE: _ClassVar[Behavior]
    BEHAVIOR_INSUFFICIENT_FUNDS: _ClassVar[Behavior]
    BEHAVIOR_ISSUER_DECLINE: _ClassVar[Behavior]
    BEHAVIOR_INVALID: _ClassVar[Behavior]
BEHAVIOR_UNSPECIFIED: Behavior
BEHAVIOR_APPROVE: Behavior
BEHAVIOR_INSUFFICIENT_FUNDS: Behavior
BEHAVIOR_ISSUER_DECLINE: Behavior
BEHAVIOR_INVALID: Behavior

class TokenizeCardRequest(_message.Message):
    __slots__ = ("integration_id", "card_number")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    CARD_NUMBER_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    card_number: str
    def __init__(self, integration_id: _Optional[str] = ..., card_number: _Optional[str] = ...) -> None: ...

class TokenizeCardResponse(_message.Message):
    __slots__ = ("token", "metadata")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    token: str
    metadata: MethodMetadata
    def __init__(self, token: _Optional[str] = ..., metadata: _Optional[_Union[MethodMetadata, _Mapping]] = ...) -> None: ...

class DescribeMethodRequest(_message.Message):
    __slots__ = ("integration_id", "token")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    token: str
    def __init__(self, integration_id: _Optional[str] = ..., token: _Optional[str] = ...) -> None: ...

class MethodMetadata(_message.Message):
    __slots__ = ("type", "fingerprint")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    type: str
    fingerprint: str
    def __init__(self, type: _Optional[str] = ..., fingerprint: _Optional[str] = ...) -> None: ...

class ResolveRequest(_message.Message):
    __slots__ = ("integration_id", "token")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    token: str
    def __init__(self, integration_id: _Optional[str] = ..., token: _Optional[str] = ...) -> None: ...

class ProcessingFixture(_message.Message):
    __slots__ = ("card_reference", "behavior")
    CARD_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    card_reference: str
    behavior: Behavior
    def __init__(self, card_reference: _Optional[str] = ..., behavior: _Optional[_Union[Behavior, str]] = ...) -> None: ...
