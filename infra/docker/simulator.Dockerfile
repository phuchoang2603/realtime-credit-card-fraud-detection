FROM ghcr.io/astral-sh/uv:python3.14-bookworm-slim

WORKDIR /app/src/simulator
ENV UV_PYTHON_DOWNLOADS=0 PYTHONDONTWRITEBYTECODE=1

COPY src/simulator/pyproject.toml src/simulator/uv.lock src/simulator/.python-version ./
RUN uv sync --locked --no-dev --no-group notebook

COPY src/simulator/simulator ./simulator
COPY src/simulator/configs ./configs
COPY contracts/data/historical-payments.odcs.yaml /app/contracts/data/historical-payments.odcs.yaml

USER 65532:65532
ENTRYPOINT ["/app/src/simulator/.venv/bin/python", "-m", "simulator.cli"]
CMD ["--config", "/app/src/simulator/configs/historical-v1.yaml"]
