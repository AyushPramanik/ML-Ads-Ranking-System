"""Generate compact test fixtures for the Go ranking service.

Trains a small CTR model on a tiny synthetic dataset and writes ``model.json``
and ``parity_samples.json`` into the Go model package's testdata directory. The
Go parity test then verifies its native scorer reproduces LightGBM's predictions
without depending on a full training run.

Run from the repository root:

    ml/.venv/bin/python ml/scripts/export_test_fixture.py
"""

from __future__ import annotations

import json
from dataclasses import replace
from pathlib import Path

from ads_ml.config import load_settings
from ads_ml.data import generate
from ads_ml.export import export_model
from ads_ml.features import FeatureSpec
from ads_ml.model import train

MODEL_FIXTURE_DIR = Path("ranking/internal/model/testdata")
SPEC_FIXTURE_DIR = Path("ranking/internal/features/testdata")
N_SAMPLES = 100


def main() -> None:
    settings = replace(
        load_settings(),
        n_users=600,
        n_ads=120,
        n_campaigns=30,
        n_impressions=8000,
        random_seed=2024,
    )
    df = generate(settings).impressions
    result = train(df, seed=2024, num_boost_round=40)

    model = export_model(result.booster, result.feature_names, result.test_features)

    sample = result.test_features[:N_SAMPLES]
    probs = result.booster.predict(sample)
    parity = {
        "samples": [
            {"features": [float(v) for v in row], "probability": float(p)}
            for row, p in zip(sample, probs, strict=True)
        ]
    }

    MODEL_FIXTURE_DIR.mkdir(parents=True, exist_ok=True)
    (MODEL_FIXTURE_DIR / "model.json").write_text(json.dumps(model))
    (MODEL_FIXTURE_DIR / "parity_samples.json").write_text(json.dumps(parity, indent=2))

    SPEC_FIXTURE_DIR.mkdir(parents=True, exist_ok=True)
    (SPEC_FIXTURE_DIR / "feature_spec.json").write_text(
        json.dumps(FeatureSpec().to_dict(), indent=2)
    )
    print(
        f"wrote model fixtures to {MODEL_FIXTURE_DIR}/ "
        f"({len(model['trees'])} trees, {N_SAMPLES} parity samples)"
    )
    print(f"wrote spec fixture to {SPEC_FIXTURE_DIR}/feature_spec.json")


if __name__ == "__main__":
    main()
