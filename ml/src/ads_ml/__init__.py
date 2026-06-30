"""ML Ads Ranking — CTR prediction training pipeline.

This package generates a synthetic ad-click dataset, engineers features, trains a
LightGBM click-through-rate model, evaluates it, and exports a portable model
representation that the Go ranking service can score natively.
"""

__version__ = "0.1.0"
