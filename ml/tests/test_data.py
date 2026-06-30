"""Tests for synthetic dataset generation."""

from __future__ import annotations

from dataclasses import replace

from ads_ml import schema
from ads_ml.config import load_settings
from ads_ml.data import generate


def _small_settings():
    return replace(
        load_settings(),
        n_users=500,
        n_ads=120,
        n_campaigns=30,
        n_impressions=5000,
        random_seed=7,
    )


def test_generate_shapes_and_columns():
    world = generate(_small_settings())
    imp = world.impressions
    assert len(imp) == 5000
    assert len(world.users) == 500
    assert len(world.ads) == 120
    assert len(world.campaigns) == 30

    expected = {
        "user_id",
        "ad_id",
        "campaign_id",
        "age",
        "gender",
        "country",
        "device",
        "hour",
        "day_of_week",
        "user_historical_ctr",
        "ad_historical_ctr",
        "campaign_budget",
        "ad_category",
        "user_ad_prior_impressions",
        "user_ad_prior_clicks",
        "position",
        schema.TARGET_COLUMN,
    }
    assert expected.issubset(set(imp.columns))


def test_generation_is_deterministic():
    a = generate(_small_settings()).impressions
    b = generate(_small_settings()).impressions
    assert a.equals(b)


def test_click_rate_and_signal_are_plausible():
    imp = generate(_small_settings()).impressions
    ctr = imp[schema.TARGET_COLUMN].mean()
    assert 0.03 < ctr < 0.40, f"unrealistic CTR {ctr}"

    # Clicks must correlate positively with the observable quality signals,
    # otherwise the dataset carries no learnable structure.
    clicked = imp[imp[schema.TARGET_COLUMN] == 1]
    not_clicked = imp[imp[schema.TARGET_COLUMN] == 0]
    assert clicked["ad_historical_ctr"].mean() > not_clicked["ad_historical_ctr"].mean()
    assert clicked["user_historical_ctr"].mean() > not_clicked["user_historical_ctr"].mean()


def test_categorical_values_are_in_vocabulary():
    imp = generate(_small_settings()).impressions
    assert set(imp["gender"].unique()).issubset(set(schema.GENDERS))
    assert set(imp["country"].unique()).issubset(set(schema.COUNTRIES))
    assert set(imp["device"].unique()).issubset(set(schema.DEVICES))
    assert set(imp["ad_category"].unique()).issubset(set(schema.AD_CATEGORIES))
