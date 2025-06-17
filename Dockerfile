FROM ghcr.io/astral-sh/uv:python3.8-bookworm

# Set working directory
WORKDIR /app

COPY ./requirements.txt .
RUN uv pip install --system -r requirements.txt

COPY ./app ./app
COPY ./models ./models

# Expose the port for the API (8000) and for Prometheus metrics (8010)
EXPOSE 8000
EXPOSE 8010

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--no-access-log"]
