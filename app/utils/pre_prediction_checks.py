import asyncio
from fastapi import HTTPException
from app.schema import TransactionFeatures
from app.utils.tracing_config import traceable  # Import the decorator


# --- 1. Terminal Control Simulation ---
@traceable
async def run_terminal_control_check(customer_id: int):
    """
    Simulates a real-time check to an issuer server for card status.
    """
    # TODO: Replace the simulation below with a real API call.
    await asyncio.sleep(0)  # This is the placeholder await.

    blocked_customers = {323, 1693, 4354, 4259, 3879, 3544, 2375}

    if customer_id in blocked_customers:
        raise HTTPException(
            status_code=403,
            detail=f"Transaction blocked by issuer: Customer account {customer_id} is under review.",
        )


# --- 2. Transaction-Blocking Rules ---
@traceable
async def run_transaction_blocking_rules(transaction: TransactionFeatures):
    """
    Applies handcrafted rules to block obvious fraud.
    """
    await asyncio.sleep(0)

    compromised_terminals = {4692, 4923, 79, 3769, 5899, 9251}

    if transaction.TERMINAL_ID in compromised_terminals:
        raise HTTPException(
            status_code=403,
            detail=f"Transaction blocked: High-risk terminal ID {transaction.TERMINAL_ID}.",
        )

    if transaction.CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW > 0:
        is_high_value = transaction.TX_AMOUNT > 100.0
        is_anomalous = transaction.TX_AMOUNT > (
            transaction.CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW * 5
        )
        if is_high_value and is_anomalous:
            raise HTTPException(
                status_code=403,
                detail="Transaction blocked: Amount is highly anomalous.",
            )
