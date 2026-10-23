import json
from pathlib import Path

import pytest
from google.protobuf.json_format import MessageToDict, ParseDict

from payments.v1.events_pb2 import PaymentEvent


@pytest.mark.parametrize(
    "fixture",
    sorted((Path(__file__).resolve().parents[3] / "contracts/payments/v1/testdata").glob("*.json")),
    ids=lambda path: path.stem,
)
def test_payment_contract_json_round_trip(fixture):
    for raw in json.loads(fixture.read_text())["events"]:
        event = ParseDict(raw, PaymentEvent())
        assert event.WhichOneof("payload") is not None
        assert ParseDict(MessageToDict(event), PaymentEvent()) == event
