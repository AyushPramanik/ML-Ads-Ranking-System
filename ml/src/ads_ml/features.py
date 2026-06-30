"""Feature engineering pipeline shared, by contract, with the Go service.

The pipeline turns raw impression rows into the numeric feature matrix the model
consumes. Categorical values are ordinal-encoded against the vocabularies in
:mod:`ads_ml.schema`; unknown values fall back to the vocabulary's last bucket.

The encodings and feature ordering are serialised to ``feature_spec.json`` so the
Go ranking service performs *identical* feature generation at inference time.
"""

from __future__ import annotations

from dataclasses import dataclass, field

import numpy as np
import pandas as pd

from . import schema


@dataclass(frozen=True)
class FeatureSpec:
    """Serialisable description of the feature contract."""

    version: int = schema.FEATURE_SPEC_VERSION
    feature_columns: list[str] = field(default_factory=lambda: list(schema.FEATURE_COLUMNS))
    categorical_vocabs: dict[str, list[str]] = field(
        default_factory=lambda: {k: list(v) for k, v in schema.CATEGORICAL_VOCABS.items()}
    )

    def to_dict(self) -> dict:
        return {
            "version": self.version,
            "feature_columns": self.feature_columns,
            "categorical_vocabs": self.categorical_vocabs,
        }


def encode_categorical(values: pd.Series, vocab: list[str]) -> np.ndarray:
    """Ordinal-encode a categorical column; unknowns map to the last bucket."""
    lookup = {label: code for code, label in enumerate(vocab)}
    fallback = len(vocab) - 1
    return values.map(lambda v: lookup.get(v, fallback)).to_numpy()


def build_features(df: pd.DataFrame) -> pd.DataFrame:
    """Produce the numeric feature matrix (columns ordered per the schema)."""
    out = pd.DataFrame(index=df.index)
    out["age"] = df["age"].astype(float)
    out["gender_code"] = encode_categorical(df["gender"], schema.GENDERS)
    out["country_code"] = encode_categorical(df["country"], schema.COUNTRIES)
    out["device_code"] = encode_categorical(df["device"], schema.DEVICES)
    out["hour"] = df["hour"].astype(float)
    out["day_of_week"] = df["day_of_week"].astype(float)
    out["user_historical_ctr"] = df["user_historical_ctr"].astype(float)
    out["ad_historical_ctr"] = df["ad_historical_ctr"].astype(float)
    out["campaign_budget"] = df["campaign_budget"].astype(float)
    out["ad_category_code"] = encode_categorical(df["ad_category"], schema.AD_CATEGORIES)
    out["user_ad_prior_impressions"] = df["user_ad_prior_impressions"].astype(float)
    out["user_ad_prior_clicks"] = df["user_ad_prior_clicks"].astype(float)
    out["position"] = df["position"].astype(float)
    return out[schema.FEATURE_COLUMNS]
