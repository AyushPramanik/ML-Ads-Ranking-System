"""Command-line entry point for the ML pipeline.

Subcommands
-----------
generate : build the synthetic dataset and write it to ``--out``.
train    : train the CTR model and export serving artifacts to ``--out``.

Run ``python -m ads_ml.cli <subcommand> --help`` for options.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import numpy as np

from . import config
from .config import Settings, load_settings
from .data import generate
from .export import export_model
from .features import FeatureSpec
from .model import train


def _ensure_dir(path: str) -> Path:
    p = Path(path)
    p.mkdir(parents=True, exist_ok=True)
    return p


def cmd_generate(args: argparse.Namespace, settings: Settings) -> int:
    out = _ensure_dir(args.out)
    world = generate(settings)
    dataset_path = out / config.DATASET_FILE
    world.impressions.to_parquet(dataset_path, index=False)
    print(
        f"Generated {len(world.impressions):,} impressions "
        f"({world.impressions['clicked'].mean():.3%} CTR) -> {dataset_path}"
    )
    print(
        f"  world: {len(world.users):,} users, {len(world.ads):,} ads, "
        f"{len(world.campaigns):,} campaigns"
    )
    return 0


def cmd_train(args: argparse.Namespace, settings: Settings) -> int:
    import pandas as pd

    data_dir = Path(args.data)
    dataset_path = data_dir / config.DATASET_FILE
    if not dataset_path.exists():
        print(
            f"dataset not found at {dataset_path}; run `generate` first",
            file=sys.stderr,
        )
        return 1

    df = pd.read_parquet(dataset_path)
    result = train(df, seed=settings.random_seed)

    out = _ensure_dir(args.out)

    # 1. Portable JSON model for the Go scorer.
    model_dict = export_model(result.booster, result.feature_names, result.test_features)
    (out / config.MODEL_FILE).write_text(json.dumps(model_dict))

    # 2. Native LightGBM model for Python-side reuse / debugging.
    result.booster.save_model(str(out / config.MODEL_NATIVE_FILE))

    # 3. Feature spec shared with the Go service.
    (out / config.FEATURE_SPEC_FILE).write_text(json.dumps(FeatureSpec().to_dict(), indent=2))

    # 4. Evaluation metrics.
    (out / config.METRICS_FILE).write_text(json.dumps(result.metrics.as_dict(), indent=2))

    # 5. Parity samples: feature vectors + expected probabilities for the Go test.
    _write_parity_samples(out, result.booster, result.test_features)

    m = result.metrics
    print(f"Trained CTR model on {m.n_train:,} rows, evaluated on {m.n_test:,} rows")
    print(
        f"  AUC={m.auc:.4f}  LogLoss={m.log_loss:.4f}  "
        f"Precision={m.precision:.4f}  Recall={m.recall:.4f}"
    )
    print(f"Artifacts written to {out}/")
    return 0


def _write_parity_samples(out: Path, booster, test_features: np.ndarray, n: int = 200) -> None:
    sample = test_features[:n]
    probs = booster.predict(sample)
    payload = {
        "samples": [
            {"features": [float(v) for v in row], "probability": float(p)}
            for row, p in zip(sample, probs, strict=True)
        ]
    }
    (out / config.PARITY_FILE).write_text(json.dumps(payload))


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="ads-ml", description="ML Ads Ranking pipeline")
    sub = parser.add_subparsers(dest="command", required=True)

    settings = load_settings()

    g = sub.add_parser("generate", help="generate the synthetic dataset")
    g.add_argument("--out", default=settings.data_dir, help="output data directory")
    g.set_defaults(func=cmd_generate)

    t = sub.add_parser("train", help="train and export the CTR model")
    t.add_argument("--data", default=settings.data_dir, help="input data directory")
    t.add_argument("--out", default=settings.artifacts_dir, help="artifact output directory")
    t.set_defaults(func=cmd_train)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    settings = load_settings()
    return args.func(args, settings)


if __name__ == "__main__":
    raise SystemExit(main())
