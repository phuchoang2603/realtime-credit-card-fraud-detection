FROM ghcr.io/astral-sh/uv:python3.14-bookworm-slim

WORKDIR /app
ENV UV_PYTHON_DOWNLOADS=0

COPY pyproject.toml uv.lock .python-version ./
RUN uv sync --locked --no-dev

COPY app ./app
COPY fraud ./fraud
COPY models ./models

EXPOSE 8000 8010

USER 65532:65532
CMD ["/app/.venv/bin/python", "-m", "app"]
