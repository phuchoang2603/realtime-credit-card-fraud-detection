FROM ghcr.io/astral-sh/uv:python3.8-bookworm

# Set working directory
WORKDIR /client

COPY ./test_client.py .

RUN uv pip install --system requests

CMD ["python", "-u", "test_client.py"]
