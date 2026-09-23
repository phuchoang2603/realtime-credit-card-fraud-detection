import json
import os
import random
import time
import uuid
from datetime import UTC, datetime
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

# Get the server URL from an environment variable, with a default for local testing
API_URL = os.environ.get("API_URL", "http://localhost:8080/predict")

# --- Lists of IDs based on the API's rules ---
BLOCKED_CUSTOMERS = [323, 1693, 4354, 4259, 3879, 3544, 2375]
COMPROMISED_TERMINALS = [4692, 4923, 79, 3769, 5899, 9251]


def generate_legitimate_data():
    """Generates a random, legitimate-looking transaction payload."""
    tx_datetime = datetime.now(UTC)
    customer_id = random.randint(10000, 20000)
    terminal_id = random.randint(10000, 20000)

    return {
        "TRANSACTION_ID": random.randint(100000, 999999),
        "TX_DATETIME": tx_datetime.isoformat(),
        "CUSTOMER_ID": customer_id,
        "TERMINAL_ID": terminal_id,
        "TX_TIME_SECONDS": int(tx_datetime.timestamp()),
        "TX_TIME_DAYS": int(tx_datetime.timestamp() / (24 * 3600)),
        "TX_AMOUNT": round(random.uniform(5.0, 150.0), 2),
        "TX_DURING_WEEKEND": 1 if tx_datetime.weekday() >= 5 else 0,
        "TX_DURING_NIGHT": 1 if 0 <= tx_datetime.hour <= 6 else 0,
        "CUSTOMER_ID_NB_TX_1DAY_WINDOW": float(random.randint(1, 5)),
        "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW": round(random.uniform(20.0, 100.0), 2),
        "CUSTOMER_ID_NB_TX_7DAY_WINDOW": float(random.randint(5, 25)),
        "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": round(random.uniform(50.0, 100.0), 2),
        "CUSTOMER_ID_NB_TX_30DAY_WINDOW": float(random.randint(20, 100)),
        "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW": round(random.uniform(70.0, 100.0), 2),
        "TERMINAL_ID_NB_TX_1DAY_WINDOW": float(random.randint(1, 100)),
        "TERMINAL_ID_RISK_1DAY_WINDOW": round(random.random(), 2),
        "TERMINAL_ID_NB_TX_7DAY_WINDOW": float(random.randint(10, 500)),
        "TERMINAL_ID_RISK_7DAY_WINDOW": round(random.random(), 2),
        "TERMINAL_ID_NB_TX_30DAY_WINDOW": float(random.randint(50, 2000)),
        "TERMINAL_ID_RISK_30DAY_WINDOW": round(random.random(), 2),
    }


# --- Fraud Scenario Generators ---


def generate_blocked_customer_fraud():
    """Generates a transaction from a known blocked customer."""
    data = generate_legitimate_data()
    data["CUSTOMER_ID"] = random.choice(BLOCKED_CUSTOMERS)
    return data


def generate_compromised_terminal_fraud():
    """Generates a transaction from a known compromised terminal."""
    data = generate_legitimate_data()
    data["TERMINAL_ID"] = random.choice(COMPROMISED_TERMINALS)
    data["TX_AMOUNT"] = round(random.uniform(300.0, 1000.0), 2)  # Also make amount high
    return data


def generate_anomalous_amount_fraud():
    """
    Generates a transaction with an amount that is anomalously high
    compared to the customer's history.
    """
    data = generate_legitimate_data()
    # Set history to a low average amount
    avg_amount = round(random.uniform(20.0, 50.0), 2)
    data["CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW"] = avg_amount
    # Set the transaction amount to be > $100 and > 5x the average
    data["TX_AMOUNT"] = avg_amount * 6
    return data


def call_predict_api():
    """Continuously calls the prediction API, cycling through fraud scenarios."""

    fraud_generators = [
        generate_blocked_customer_fraud,
        generate_compromised_terminal_fraud,
        generate_anomalous_amount_fraud,
    ]

    while True:
        try:
            # 25% chance to generate a fraudulent transaction
            if random.random() < 0.25:
                # Randomly pick one of the three fraud scenarios
                fraud_generator = random.choice(fraud_generators)
                print(f">>> Generating FRAUD transaction via: {fraud_generator.__name__}")
                transaction_data = fraud_generator()
            else:
                print(">>> Generating a legitimate transaction...")
                transaction_data = generate_legitimate_data()

            headers = {"X-Request-ID": str(uuid.uuid4())}
            print(f"Sending request to {API_URL}...")

            body = json.dumps({key.lower(): value for key, value in transaction_data.items()}).encode()
            request = Request(API_URL, data=body, headers=headers | {"Content-Type": "application/json"})
            with urlopen(request, timeout=5) as response:
                print(f"Response: {response.read().decode()}")
        except HTTPError as error:
            print(f"HTTP {error.code}: {error.read().decode()}")
        except URLError as error:
            print(f"Could not connect to edge: {error.reason}")

        time.sleep(2)


if __name__ == "__main__":
    call_predict_api()
