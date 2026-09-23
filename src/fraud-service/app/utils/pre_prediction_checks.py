from app.errors import AnomalousAmountError, BlockedCustomerError, CompromisedTerminalError
from app.schema import TransactionFeatures

BLOCKED_CUSTOMERS = frozenset({323, 1693, 4354, 4259, 3879, 3544, 2375})
COMPROMISED_TERMINALS = frozenset({4692, 4923, 79, 3769, 5899, 9251})


def check_customer(customer_id: int) -> None:
    if customer_id in BLOCKED_CUSTOMERS:
        raise BlockedCustomerError(customer_id)


def check_transaction(transaction: TransactionFeatures) -> None:
    if transaction.TERMINAL_ID in COMPROMISED_TERMINALS:
        raise CompromisedTerminalError(transaction.TERMINAL_ID)
    if (
        transaction.CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW > 0
        and transaction.TX_AMOUNT > 100
        and transaction.TX_AMOUNT > transaction.CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW * 5
    ):
        raise AnomalousAmountError
