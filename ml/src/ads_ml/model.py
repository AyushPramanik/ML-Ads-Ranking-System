"""CTR model training, evaluation, and serialization."""

from __future__ import annotations

from dataclasses import asdict, dataclass

import lightgbm as lgb
import numpy as np
import pandas as pd
from sklearn.metrics import (
    log_loss,
    precision_score,
    recall_score,
    roc_auc_score,
)
from sklearn.model_selection import train_test_split

from . import schema
from .features import build_features


@dataclass
class EvalMetrics:
    """Headline classification metrics for the CTR model."""

    auc: float
    log_loss: float
    precision: float
    recall: float
    threshold: float
    n_train: int
    n_test: int
    positive_rate: float

    def as_dict(self) -> dict:
        return asdict(self)


@dataclass
class TrainResult:
    booster: lgb.Booster
    metrics: EvalMetrics
    feature_names: list[str]
    test_features: np.ndarray


def _default_params() -> dict:
    return {
        "objective": "binary",
        "boosting_type": "gbdt",
        "learning_rate": 0.05,
        "num_leaves": 31,
        "max_depth": -1,
        "min_child_samples": 50,
        "feature_fraction": 0.9,
        "bagging_fraction": 0.8,
        "bagging_freq": 1,
        "verbosity": -1,
    }


def train(
    df: pd.DataFrame,
    *,
    seed: int = 42,
    num_boost_round: int = 300,
    test_size: float = 0.2,
    threshold: float | None = None,
) -> TrainResult:
    """Train the CTR model and compute evaluation metrics on a held-out split.

    When ``threshold`` is ``None`` the decision threshold defaults to the dataset's
    positive rate. For heavily imbalanced CTR data this is a far more informative
    operating point than 0.5, yielding balanced precision/recall while AUC and log
    loss remain threshold-independent measures of ranking quality.
    """
    features = build_features(df)
    target = df[schema.TARGET_COLUMN].astype(int).to_numpy()
    if threshold is None:
        threshold = float(np.mean(target))

    x_train, x_test, y_train, y_test = train_test_split(
        features.to_numpy(),
        target,
        test_size=test_size,
        random_state=seed,
        stratify=target,
    )

    params = _default_params()
    params["seed"] = seed
    train_set = lgb.Dataset(x_train, label=y_train, feature_name=schema.FEATURE_COLUMNS)
    booster = lgb.train(params, train_set, num_boost_round=num_boost_round)

    probs = booster.predict(x_test)
    preds = (probs >= threshold).astype(int)
    metrics = EvalMetrics(
        auc=float(roc_auc_score(y_test, probs)),
        log_loss=float(log_loss(y_test, probs)),
        precision=float(precision_score(y_test, preds, zero_division=0)),
        recall=float(recall_score(y_test, preds, zero_division=0)),
        threshold=threshold,
        n_train=int(len(y_train)),
        n_test=int(len(y_test)),
        positive_rate=float(np.mean(target)),
    )

    return TrainResult(
        booster=booster,
        metrics=metrics,
        feature_names=list(schema.FEATURE_COLUMNS),
        test_features=x_test,
    )
