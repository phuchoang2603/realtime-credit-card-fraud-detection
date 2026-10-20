FROM ghcr.io/astral-sh/uv:python3.14-bookworm-slim

WORKDIR /app
ENV UV_PYTHON_DOWNLOADS=0 PYTHONPATH=/app/gen PYTHONDONTWRITEBYTECODE=1

COPY pyproject.toml uv.lock .python-version ./
RUN uv sync --locked --no-dev --no-group notebook

COPY simulator ./simulator
COPY gen ./gen
COPY configs ./configs

USER 65532:65532
ENTRYPOINT ["/app/.venv/bin/python", "-m", "simulator.cli"]
CMD ["--config", "/app/configs/historical-v1.yaml"]
