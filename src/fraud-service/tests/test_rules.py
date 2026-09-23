import pytest

from app.errors import AnomalousAmountError, BlockedCustomerError, CompromisedTerminalError
from app.schema import TransactionFeatures
from app.utils.pre_prediction_checks import check_customer, check_transaction


def transaction(payload):
    return TransactionFeatures.model_validate(payload)


@pytest.mark.parametrize(
    ("amount", "raises"),
    [(100.0, False), (100.01, True)],
    ids=["at-high-value-boundary", "above-high-value-boundary"],
)
def test_high_value_boundary(sample_legitimate_payload, amount, raises):
    payload = sample_legitimate_payload | {"TX_AMOUNT": amount, "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": 10.0}

    def call():
        check_transaction(transaction(payload))

    if raises:
        with pytest.raises(AnomalousAmountError):
            call()
    else:
        call()


@pytest.mark.parametrize(
    ("average", "raises"),
    [(30.0, False), (29.99, True), (1.0, True), (0.0, False)],
    ids=["at-five-times-boundary", "above-five-times-boundary", "positive-low-average", "zero-average-partition"],
)
def test_anomaly_ratio_boundary(sample_legitimate_payload, average, raises):
    payload = sample_legitimate_payload | {"TX_AMOUNT": 150.0, "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": average}

    def call():
        check_transaction(transaction(payload))

    if raises:
        with pytest.raises(AnomalousAmountError):
            call()
    else:
        call()


@pytest.mark.parametrize(
    ("customer_id", "expected_error"),
    [(1001, False), (323, True)],
    ids=["allowed-customer", "blocked-customer"],
)
def test_customer_partition(customer_id, expected_error):
    def call():
        check_customer(customer_id)

    if expected_error:
        with pytest.raises(BlockedCustomerError):
            call()
    else:
        call()


@pytest.mark.parametrize("terminal_id", [2001, 4692], ids=["allowed-terminal", "compromised-terminal"])
def test_terminal_partition(sample_legitimate_payload, terminal_id):
    payload = sample_legitimate_payload | {"TERMINAL_ID": terminal_id}

    def call():
        check_transaction(transaction(payload))

    if terminal_id == 4692:
        with pytest.raises(CompromisedTerminalError):
            call()
    else:
        call()
