"""Tests for model training, evaluation, and the portable export.

The export parity test is the linchpin of cross-language correctness: it proves
the flattened tree representation reproduces LightGBM's own predictions, so the
Go scorer that consumes the same structure will agree with Python.
"""

from __future__ import annotations

from dataclasses import replace

import numpy as np

from ads_ml.config import load_settings
from ads_ml.data import generate
from ads_ml.export import _leaf_sum, export_model
from ads_ml.model import train


def _train_small():
    settings = replace(
        load_settings(),
        n_users=800,
        n_ads=150,
        n_campaigns=40,
        n_impressions=12000,
        random_seed=11,
    )
    df = generate(settings).impressions
    return train(df, seed=11, num_boost_round=60)


def test_metrics_indicate_learnable_signal():
    result = _train_small()
    m = result.metrics
    assert m.auc > 0.65, f"AUC too low: {m.auc}"
    assert 0.0 < m.log_loss < 1.0
    assert m.n_train + m.n_test == 12000


def test_export_reproduces_lightgbm_predictions():
    result = _train_small()
    model = export_model(result.booster, result.feature_names, result.test_features)

    trees = [t["nodes"] for t in model["trees"]]
    sample = result.test_features[:100]
    expected = result.booster.predict(sample)

    for row, exp in zip(sample, expected, strict=True):
        raw = model["base_score"] + _leaf_sum(trees, row)
        prob = 1.0 / (1.0 + np.exp(-raw))
        assert abs(prob - exp) < 1e-9


def test_export_structure_is_well_formed():
    result = _train_small()
    model = export_model(result.booster, result.feature_names, result.test_features)
    assert model["objective"] == "binary"
    assert model["num_features"] == len(result.feature_names)
    assert model["feature_names"] == result.feature_names
    assert len(model["trees"]) > 0
    # Every internal node must reference valid child indices.
    for tree in model["trees"]:
        nodes = tree["nodes"]
        for node in nodes:
            if not node["leaf"]:
                assert 0 <= node["left"] < len(nodes)
                assert 0 <= node["right"] < len(nodes)
