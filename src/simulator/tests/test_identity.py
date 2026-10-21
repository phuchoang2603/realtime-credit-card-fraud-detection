import json
from pathlib import Path
from uuid import UUID

from simulator.identity import IDENTITY_NAMESPACE, entity_id, fingerprint

VECTORS = json.loads((Path(__file__).resolve().parents[3] / "contracts/synthetic/identity-vectors.json").read_text())


def test_identity_vectors():
    assert UUID(VECTORS["uuid_namespace"]) == IDENTITY_NAMESPACE
    integration = UUID(VECTORS["integration_id"])
    for case in VECTORS["ids"]:
        assert entity_id(integration, case["kind"], case["external_id"]) == case["expected"]
    for case in VECTORS["fingerprints"]:
        assert fingerprint(VECTORS["synthetic_key"], case["kind"], case["input"]) == case["expected"]
