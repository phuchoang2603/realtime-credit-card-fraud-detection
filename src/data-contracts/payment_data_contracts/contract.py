from dataclasses import dataclass
from hashlib import sha256
from importlib.resources import files
from pathlib import Path
from typing import Any

import pyarrow as pa
import yaml
from open_data_contract_standard.model import OpenDataContractStandard, SchemaObject


class _ContractLoader(yaml.SafeLoader):
    def construct_mapping(self, node, deep=False):
        self.flatten_mapping(node)
        keys = [self.construct_object(key, deep=deep) for key, _ in node.value]
        if len(keys) != len(set(keys)):
            raise ValueError("duplicate YAML keys are not allowed in contracts")
        return super().construct_mapping(node, deep=deep)


def custom_property(element: Any, name: str, default: Any = None) -> Any:
    values = [item.value for item in element.customProperties or [] if item.property == name]
    if len(values) > 1:
        raise ValueError(f"duplicate custom property: {name}")
    return values[0] if values else default


def _arrow_type(prop: Any, version: int, versions: tuple[int, ...]) -> pa.DataType:
    if prop.logicalType == "object":
        if prop.physicalType is not None:
            raise ValueError("object physical types are derived from their properties")
        return pa.struct(_arrow_fields(prop.properties or [], version, versions))
    if prop.logicalType == "array" and prop.items is not None:
        if prop.physicalType is not None:
            raise ValueError("array physical types are derived from their items")
        return pa.list_(
            pa.field("element", _arrow_type(prop.items, version, versions), nullable=not prop.items.required)
        )
    if prop.logicalType == "timestamp" and prop.logicalTypeOptions != {"timezone": True, "defaultTimezone": "UTC"}:
        raise ValueError("timestamp mapping requires a UTC timezone")
    types = {
        ("string", "VARCHAR"): pa.string(),
        ("integer", "BIGINT"): pa.int64(),
        ("integer", "INTEGER"): pa.int32(),
        ("integer", "UINTEGER"): pa.uint32(),
        ("number", "DOUBLE"): pa.float64(),
        ("boolean", "BOOLEAN"): pa.bool_(),
        ("timestamp", "TIMESTAMPTZ"): pa.timestamp("us", tz="UTC"),
    }
    try:
        return types[prop.logicalType, prop.physicalType]
    except KeyError as exc:
        raise ValueError(f"unsupported Arrow type: {prop.logicalType}/{prop.physicalType}") from exc


def _arrow_fields(properties: list, version: int, versions: tuple[int, ...]) -> list[pa.Field]:
    fields = []
    names = [prop.name for prop in properties]
    if len(names) != len(set(names)):
        raise ValueError("duplicate property names")
    for prop in properties:
        if prop.physicalName not in (None, prop.name):
            raise ValueError(f"physicalName aliases are not supported: {prop.name}")
        since = custom_property(prop, "sinceSchemaVersion", versions[0])
        if type(since) is not int or since not in versions:
            raise ValueError(f"unsupported sinceSchemaVersion: {since}")
        field_type = _arrow_type(prop, version, versions)
        if version >= since:
            fields.append(pa.field(prop.name, field_type, nullable=not prop.required))
    return fields


@dataclass(frozen=True)
class Contract:
    model: OpenDataContractStandard
    sha256: str
    source: str

    def dataset(self, name: str) -> SchemaObject:
        for dataset in self.model.schema_ or []:
            if dataset.name == name:
                return dataset
        raise ValueError(f"unknown dataset: {name}")

    def versions(self, name: str) -> tuple[int, ...]:
        versions = custom_property(self.dataset(name), "schemaVersions")
        if (
            not isinstance(versions, list)
            or not versions
            or any(type(version) is not int or version < 1 for version in versions)
            or sorted(set(versions)) != versions
        ):
            raise ValueError(f"invalid schemaVersions: {name}")
        return tuple(versions)

    def arrow_schema(self, name: str, version: int | None = None) -> pa.Schema:
        versions = self.versions(name)
        version = versions[-1] if version is None else version
        if type(version) is not int or version not in versions:
            raise ValueError(f"unsupported {name} schema version: {version}")
        return pa.schema(_arrow_fields(self.dataset(name).properties or [], version, versions))

    @property
    def provenance(self) -> dict:
        return {"id": self.model.id, "version": self.model.version, "sha256": self.sha256}


def load_contract(path: Path | None = None) -> Contract:
    content = (
        path.read_bytes()
        if path is not None
        else files(__package__).joinpath("historical-payments.odcs.yaml").read_bytes()
    )
    model = OpenDataContractStandard.model_validate(yaml.load(content, Loader=_ContractLoader))
    if model.apiVersion != "v3.1.0":
        raise ValueError(f"unsupported ODCS version: {model.apiVersion}")
    return Contract(model, sha256(content).hexdigest(), content.decode())
