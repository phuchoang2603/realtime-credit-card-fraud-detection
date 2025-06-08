FROM ghcr.io/astral-sh/uv:python3.8-bookworm

# Set working directory
WORKDIR /app

COPY ./app /app
COPY ./requirements.txt /app
COPY ./models /app/models

ENV MODEL_PATH=/app/models/model.pkl
ENV LOG_LEVEL=INFO

RUN uv pip install --system -r requirements.txt

CMD ["python", "main.py"]
