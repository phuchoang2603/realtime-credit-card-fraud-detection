import pytest
from pydantic import ValidationError

from simulator.cli import destination
from tests.conftest import small_config


def test_unknown_scenario_is_rejected():
    with pytest.raises(ValidationError, match="device_farm"):
        small_config(**{"scenarios.device_farm": {"rate": 0.1}})


def test_credential_fields_are_rejected():
    with pytest.raises(ValidationError, match="aws_secret_access_key"):
        small_config(aws_secret_access_key="secret")


def test_cutover_must_fall_inside_the_dataset():
    with pytest.raises(ValidationError, match="schema_cutover_date"):
        small_config(**{"data_problems.schema_cutover_date": "2027-01-01"})


def test_output_url_must_not_carry_credentials():
    with pytest.raises(ValueError, match="without credentials"):
        destination("s3://key:secret@synthetic-source")
