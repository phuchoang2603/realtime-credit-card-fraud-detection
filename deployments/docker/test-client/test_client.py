import os
import requests
import random
import time
import uuid

# Get the server URL from an environment variable, with a default for local testing
API_URL = os.environ.get("API_URL", "http://localhost:8000/predict")


def generate_transaction_data():
    """Generates a random transaction payload for the API."""
    return {
        "TX_AMOUNT": round(random.uniform(5.0, 500.0), 2),
        "TX_DURING_WEEKEND": random.choice([0, 1]),
        "TX_DURING_NIGHT": random.choice([0, 1]),
        "CUSTOMER_ID_NB_TX_1DAY_WINDOW": float(random.randint(1, 10)),
        "CUSTOMER_ID_AVG_AMOUNT_1DAY_WINDOW": round(random.uniform(20.0, 400.0), 2),
        "CUSTOMER_ID_NB_TX_7DAY_WINDOW": float(random.randint(5, 50)),
        "CUSTOMER_ID_AVG_AMOUNT_7DAY_WINDOW": round(random.uniform(50.0, 350.0), 2),
        "CUSTOMER_ID_NB_TX_30DAY_WINDOW": float(random.randint(20, 200)),
        "CUSTOMER_ID_AVG_AMOUNT_30DAY_WINDOW": round(random.uniform(70.0, 300.0), 2),
        "TERMINAL_ID_NB_TX_1DAY_WINDOW": float(random.randint(1, 100)),
        "TERMINAL_ID_RISK_1DAY_WINDOW": round(random.random(), 2),
        "TERMINAL_ID_NB_TX_7DAY_WINDOW": float(random.randint(10, 500)),
        "TERMINAL_ID_RISK_7DAY_WINDOW": round(random.random(), 2),
        "TERMINAL_ID_NB_TX_30DAY_WINDOW": float(random.randint(50, 2000)),
        "TERMINAL_ID_RISK_30DAY_WINDOW": round(random.random(), 2),
    }


def call_predict_api():
    """Continuously calls the prediction API with random data."""
    while True:
        try:
            transaction_data = generate_transaction_data()
            headers = {"X-Request-ID": str(uuid.uuid4())}

            print(f"Sending request to {API_URL}...")
            # print(f"Payload: {transaction_data}")

            response = requests.post(API_URL, json=transaction_data, headers=headers)
            response.raise_for_status()  # Raise an exception for bad status codes (4xx or 5xx)

            print(f"Request successful!")
            print(f"  Response: {response.json()}")
            print("-" * 30)

        except requests.exceptions.RequestException as e:
            print(f"Could not connect to the API: {e}")
            print("-" * 30)

        # Wait for a short period before sending the next request
        time.sleep(2)


if __name__ == "__main__":
    call_predict_api()
