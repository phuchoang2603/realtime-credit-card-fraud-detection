from datetime import timezone

import pyarrow as pa
from google.protobuf.descriptor import Descriptor, FieldDescriptor
from google.protobuf.message import Message


def arrow_type(field: FieldDescriptor) -> pa.DataType:
    if field.type == FieldDescriptor.TYPE_MESSAGE:
        if field.message_type.full_name == "google.protobuf.Timestamp":
            result = pa.timestamp("us", tz="UTC")
        elif not field.message_type.fields:
            result = pa.struct([pa.field("_present", pa.bool_())])
        else:
            result = pa.struct([pa.field(child.name, arrow_type(child)) for child in field.message_type.fields])
    elif field.type == FieldDescriptor.TYPE_ENUM:
        result = pa.string()
    elif field.type in (FieldDescriptor.TYPE_INT32, FieldDescriptor.TYPE_SINT32, FieldDescriptor.TYPE_SFIXED32):
        result = pa.int32()
    elif field.type in (FieldDescriptor.TYPE_UINT32, FieldDescriptor.TYPE_FIXED32):
        result = pa.uint32()
    elif field.type in (
        FieldDescriptor.TYPE_INT64,
        FieldDescriptor.TYPE_UINT64,
        FieldDescriptor.TYPE_SINT64,
        FieldDescriptor.TYPE_FIXED64,
        FieldDescriptor.TYPE_SFIXED64,
    ):
        result = pa.int64()
    elif field.type == FieldDescriptor.TYPE_BOOL:
        result = pa.bool_()
    elif field.type == FieldDescriptor.TYPE_FLOAT:
        result = pa.float32()
    elif field.type == FieldDescriptor.TYPE_DOUBLE:
        result = pa.float64()
    elif field.type == FieldDescriptor.TYPE_STRING:
        result = pa.string()
    elif field.type == FieldDescriptor.TYPE_BYTES:
        result = pa.binary()
    else:
        raise TypeError(f"unsupported protobuf field type: {field.full_name}")
    if field.is_repeated:
        return pa.list_(result)
    return result


def arrow_schema(descriptor: Descriptor) -> pa.Schema:
    return pa.schema([pa.field(field.name, arrow_type(field)) for field in descriptor.fields])


def message_row(message: Message) -> dict:
    if not message.DESCRIPTOR.fields:
        return {"_present": True}
    row = {}
    for field in message.DESCRIPTOR.fields:
        if field.has_presence and not message.HasField(field.name):
            row[field.name] = None
            continue
        value = getattr(message, field.name)
        if field.is_repeated:
            row[field.name] = [_field_value(field, item) for item in value]
        else:
            row[field.name] = _field_value(field, value)
    return row


def _field_value(field: FieldDescriptor, value):
    if field.type == FieldDescriptor.TYPE_MESSAGE:
        if field.message_type.full_name == "google.protobuf.Timestamp":
            return value.ToDatetime(tzinfo=timezone.utc)
        return message_row(value)
    if field.type == FieldDescriptor.TYPE_ENUM:
        return field.enum_type.values_by_number[value].name
    return value


def table_from_messages(messages: list[Message]) -> pa.Table:
    if not messages:
        raise ValueError("at least one message is required to build an Arrow table")
    return pa.Table.from_pylist(
        [message_row(message) for message in messages], schema=arrow_schema(messages[0].DESCRIPTOR)
    )
