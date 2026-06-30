"""Tests for the feature engineering pipeline and feature spec."""

from __future__ import annotations

import pandas as pd

from ads_ml import schema
from ads_ml.features import FeatureSpec, build_features, encode_categorical


def test_encode_categorical_known_values():
    s = pd.Series(["female", "male", "unknown"])
    codes = encode_categorical(s, schema.GENDERS)
    assert list(codes) == [0, 1, 2]


def test_encode_categorical_unknown_falls_back_to_last_bucket():
    s = pd.Series(["US", "ATLANTIS", "GB"])
    codes = encode_categorical(s, schema.COUNTRIES)
    fallback = len(schema.COUNTRIES) - 1
    assert list(codes) == [0, fallback, 1]


def test_build_features_column_order_matches_schema():
    df = pd.DataFrame(
        {
            "age": [30],
            "gender": ["male"],
            "country": ["US"],
            "device": ["mobile"],
            "hour": [18],
            "day_of_week": [5],
            "user_historical_ctr": [0.1],
            "ad_historical_ctr": [0.05],
            "campaign_budget": [1000.0],
            "ad_category": ["retail"],
            "user_ad_prior_impressions": [3],
            "user_ad_prior_clicks": [1],
            "position": [0],
        }
    )
    feats = build_features(df)
    assert list(feats.columns) == schema.FEATURE_COLUMNS
    row = feats.iloc[0]
    assert row["gender_code"] == 1
    assert row["country_code"] == 0
    assert row["device_code"] == 0
    assert row["ad_category_code"] == 0


def test_feature_spec_roundtrips_to_dict():
    spec = FeatureSpec().to_dict()
    assert spec["version"] == schema.FEATURE_SPEC_VERSION
    assert spec["feature_columns"] == schema.FEATURE_COLUMNS
    assert spec["categorical_vocabs"]["device"] == schema.DEVICES
