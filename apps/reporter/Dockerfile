FROM python:3.12-slim AS build
ENV POETRY_VERSION=1.8.3 \
    POETRY_VIRTUALENVS_CREATE=false \
    PIP_NO_CACHE_DIR=1
WORKDIR /app
RUN pip install "poetry==${POETRY_VERSION}"
COPY pyproject.toml poetry.lock ./
COPY src ./src
COPY README.md ./
RUN poetry install --only main --no-interaction

FROM python:3.12-slim
WORKDIR /app
COPY --from=build /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=build /usr/local/bin /usr/local/bin
COPY --from=build /app/src ./src
ENV PYTHONPATH=/app/src
ENTRYPOINT ["python", "-m", "govgate_reporter.cli"]
