# Training pipeline image. Runs as a one-shot job that generates the synthetic
# dataset, trains the CTR model, and writes serving artifacts to a shared volume
# the Go ranking service consumes.
FROM python:3.12-slim

# libgomp1 is required by LightGBM's OpenMP runtime.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install dependencies first for better layer caching.
COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .

ENV ADS_ML_DATA_DIR=/data \
    ADS_ML_ARTIFACTS_DIR=/artifacts

# Default: run the full pipeline (generate dataset, then train + export).
CMD ["sh", "-c", "ads-ml generate && ads-ml train"]
