"""Single source of truth for the feature contract.

Both the Python training pipeline and the Go ranking service must agree on the
feature ordering and categorical encodings. The training pipeline serialises the
encodings derived from this module into ``feature_spec.json``; the Go service
loads that file at startup. Keeping the vocabularies here means there is exactly
one place to change when the feature set evolves.
"""

from __future__ import annotations

# --- Categorical vocabularies ------------------------------------------------
# The final entry of each vocabulary acts as the fallback bucket for values the
# model has never seen (e.g. a new country). Encoders map unknown values to it.
GENDERS: list[str] = ["female", "male", "unknown"]
COUNTRIES: list[str] = ["US", "GB", "IN", "DE", "BR", "JP", "NG", "other"]
DEVICES: list[str] = ["mobile", "desktop", "tablet"]
AD_CATEGORIES: list[str] = [
    "retail",
    "finance",
    "gaming",
    "travel",
    "food",
    "auto",
    "tech",
    "health",
]

# Categorical features are ordinal-encoded to integer codes. This keeps every
# tree split a simple numeric threshold, which the Go scorer evaluates exactly.
CATEGORICAL_VOCABS: dict[str, list[str]] = {
    "gender": GENDERS,
    "country": COUNTRIES,
    "device": DEVICES,
    "ad_category": AD_CATEGORIES,
}

# --- Feature ordering --------------------------------------------------------
# The order here defines the index of each value in the model's input vector.
# It MUST NOT be reordered without retraining and re-exporting the model.
FEATURE_COLUMNS: list[str] = [
    "age",
    "gender_code",
    "country_code",
    "device_code",
    "hour",
    "day_of_week",
    "user_historical_ctr",
    "ad_historical_ctr",
    "campaign_budget",
    "ad_category_code",
    "user_ad_prior_impressions",
    "user_ad_prior_clicks",
    "position",
]

TARGET_COLUMN: str = "clicked"

# Feature-spec format version, bumped on any breaking change to the contract.
FEATURE_SPEC_VERSION: int = 1
