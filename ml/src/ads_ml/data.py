"""Synthetic ad-click dataset generation.

Real impression/click logs are rarely available outside production, so we
simulate them. The generator builds a small relational world — users, campaigns,
ads — then samples impressions with context (device, time, position) and draws a
click from a latent logistic model. Because the click probability is a known
function of the features, a well-specified model can recover real signal and the
evaluation metrics are meaningful rather than noise.

The simulation deliberately injects observation noise into the "historical CTR"
features so they are informative but imperfect, mirroring production where such
aggregates are stale and sparse.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd

from . import schema
from .config import Settings


@dataclass
class World:
    """The generated relational entities, exposed for inspection and seeding."""

    users: pd.DataFrame
    campaigns: pd.DataFrame
    ads: pd.DataFrame
    impressions: pd.DataFrame


def _sigmoid(x: np.ndarray) -> np.ndarray:
    return 1.0 / (1.0 + np.exp(-x))


def _make_users(rng: np.random.Generator, n: int) -> pd.DataFrame:
    """Users carry demographics and a latent propensity to click anything."""
    age = np.clip(rng.normal(38, 14, n), 13, 90).astype(int)
    gender = rng.choice(schema.GENDERS, size=n, p=[0.47, 0.47, 0.06])
    country = rng.choice(
        schema.COUNTRIES,
        size=n,
        p=[0.30, 0.10, 0.18, 0.08, 0.09, 0.07, 0.06, 0.12],
    )
    # Latent click propensity in roughly [0.02, 0.35]; never observed directly.
    propensity = rng.beta(2.0, 18.0, n) + 0.02
    return pd.DataFrame(
        {
            "user_id": np.arange(n),
            "age": age,
            "gender": gender,
            "country": country,
            "propensity": propensity,
        }
    )


def _make_campaigns(rng: np.random.Generator, n: int) -> pd.DataFrame:
    """Campaigns own a budget and an ad category."""
    category = rng.choice(schema.AD_CATEGORIES, size=n)
    # Daily budget in dollars, heavy-tailed: most modest, a few very large.
    budget = np.round(rng.lognormal(mean=7.0, sigma=1.0, size=n), 2)
    return pd.DataFrame(
        {
            "campaign_id": np.arange(n),
            "category": category,
            "budget": budget,
            "status": rng.choice(["active", "paused"], size=n, p=[0.85, 0.15]),
        }
    )


def _make_ads(rng: np.random.Generator, n: int, campaigns: pd.DataFrame) -> pd.DataFrame:
    """Ads belong to a campaign and have a latent intrinsic quality."""
    campaign_id = rng.integers(0, len(campaigns), size=n)
    quality = rng.beta(2.0, 30.0, n) + 0.01  # latent CTR ~ [0.01, 0.2]
    return pd.DataFrame(
        {
            "ad_id": np.arange(n),
            "campaign_id": campaign_id,
            "category": campaigns.loc[campaign_id, "category"].to_numpy(),
            "budget": campaigns.loc[campaign_id, "budget"].to_numpy(),
            "quality": quality,
        }
    )


def _category_effect(categories: np.ndarray) -> np.ndarray:
    """Some verticals convert better than others (additive logit shift)."""
    effect = {
        "retail": 0.25,
        "finance": -0.30,
        "gaming": 0.45,
        "travel": 0.10,
        "food": 0.30,
        "auto": -0.15,
        "tech": 0.05,
        "health": -0.10,
    }
    return np.array([effect[c] for c in categories])


def generate(settings: Settings) -> World:
    """Generate the full synthetic world and labelled impression table."""
    rng = np.random.default_rng(settings.random_seed)

    users = _make_users(rng, settings.n_users)
    campaigns = _make_campaigns(rng, settings.n_campaigns)
    ads = _make_ads(rng, settings.n_ads, campaigns)

    n = settings.n_impressions
    u_idx = rng.integers(0, len(users), size=n)
    a_idx = rng.integers(0, len(ads), size=n)

    user = users.iloc[u_idx].reset_index(drop=True)
    ad = ads.iloc[a_idx].reset_index(drop=True)

    # --- Context features ----------------------------------------------------
    device = rng.choice(schema.DEVICES, size=n, p=[0.62, 0.30, 0.08])
    hour = rng.integers(0, 24, size=n)
    day_of_week = rng.integers(0, 7, size=n)
    # Lower ranked positions get less attention.
    position = rng.integers(0, 10, size=n)

    # --- Observed (noisy) historical aggregates ------------------------------
    user_hist_ctr = np.clip(user["propensity"].to_numpy() + rng.normal(0, 0.02, n), 0.0, 1.0)
    ad_hist_ctr = np.clip(ad["quality"].to_numpy() + rng.normal(0, 0.015, n), 0.0, 1.0)

    # --- Per (user, ad) interaction history ----------------------------------
    prior_impressions = rng.poisson(1.2, size=n)
    pair_ctr = user["propensity"].to_numpy() * (1.0 + 4.0 * ad["quality"].to_numpy())
    prior_clicks = rng.binomial(prior_impressions, np.clip(pair_ctr, 0.0, 1.0))

    # --- Latent click model --------------------------------------------------
    # Effects on the log-odds scale; signal dominated by latent propensity and
    # quality, with realistic context modifiers.
    budget_norm = np.log1p(ad["budget"].to_numpy()) - 7.0
    logit = (
        -2.4
        + 9.0 * (user["propensity"].to_numpy() - 0.10)
        + 13.0 * (ad["quality"].to_numpy() - 0.06)
        + _category_effect(ad["category"].to_numpy())
        + np.where(device == "mobile", 0.20, 0.0)
        + np.where(device == "tablet", -0.10, 0.0)
        + 0.35 * np.sin((hour - 6) / 24.0 * 2 * np.pi)  # daily rhythm, peaks evening
        + np.where(day_of_week >= 5, 0.15, 0.0)  # weekend lift
        - 0.05 * position  # position decay
        + 0.015 * (user["age"].to_numpy() - 38)  # mild age tilt
        + 0.10 * budget_norm  # better-funded campaigns target better
        + 0.30 * prior_clicks  # prior engagement predicts future clicks
    )
    prob = _sigmoid(logit)
    clicked = rng.binomial(1, prob)

    impressions = pd.DataFrame(
        {
            "user_id": user["user_id"].to_numpy(),
            "ad_id": ad["ad_id"].to_numpy(),
            "campaign_id": ad["campaign_id"].to_numpy(),
            # Raw feature inputs (encoded later by the feature pipeline).
            "age": user["age"].to_numpy(),
            "gender": user["gender"].to_numpy(),
            "country": user["country"].to_numpy(),
            "device": device,
            "hour": hour,
            "day_of_week": day_of_week,
            "user_historical_ctr": user_hist_ctr,
            "ad_historical_ctr": ad_hist_ctr,
            "campaign_budget": ad["budget"].to_numpy(),
            "ad_category": ad["category"].to_numpy(),
            "user_ad_prior_impressions": prior_impressions,
            "user_ad_prior_clicks": prior_clicks,
            "position": position,
            schema.TARGET_COLUMN: clicked,
        }
    )

    return World(users=users, campaigns=campaigns, ads=ads, impressions=impressions)
