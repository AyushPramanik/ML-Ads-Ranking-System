"""Runtime configuration for the training pipeline.

Values come from environment variables with sensible defaults so the pipeline is
reproducible out of the box and configurable in CI or containers.
"""

from __future__ import annotations

import os
from dataclasses import dataclass

# Canonical artifact filenames shared with the Go service.
DATASET_FILE = "impressions.parquet"
MODEL_FILE = "model.json"
MODEL_NATIVE_FILE = "model.txt"
FEATURE_SPEC_FILE = "feature_spec.json"
METRICS_FILE = "metrics.json"
PARITY_FILE = "parity_samples.json"


@dataclass(frozen=True)
class Settings:
    """Pipeline settings resolved from the environment."""

    data_dir: str = os.getenv("ADS_ML_DATA_DIR", "./data")
    artifacts_dir: str = os.getenv("ADS_ML_ARTIFACTS_DIR", "./artifacts")
    random_seed: int = int(os.getenv("ADS_ML_RANDOM_SEED", "42"))

    # Dataset size knobs (override via env for quick local runs).
    n_users: int = int(os.getenv("ADS_ML_N_USERS", "5000"))
    n_ads: int = int(os.getenv("ADS_ML_N_ADS", "800"))
    n_campaigns: int = int(os.getenv("ADS_ML_N_CAMPAIGNS", "120"))
    n_impressions: int = int(os.getenv("ADS_ML_N_IMPRESSIONS", "200000"))


def load_settings() -> Settings:
    """Return settings resolved from the current environment."""
    return Settings()
