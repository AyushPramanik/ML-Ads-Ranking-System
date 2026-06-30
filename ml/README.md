# ads-ml — CTR training pipeline

Python package that generates a synthetic ad-click dataset, engineers features,
trains a LightGBM click-through-rate model, evaluates it, and exports a portable
model the Go ranking service scores natively.

## Layout

```
src/ads_ml/
├── schema.py     # single source of truth: feature order + categorical vocabularies
├── config.py     # environment-driven settings (paths, seed, dataset sizes)
├── data.py       # synthetic users/campaigns/ads + labelled impressions
├── features.py   # feature engineering pipeline + serialisable FeatureSpec
├── model.py      # LightGBM training + AUC/LogLoss/Precision/Recall evaluation
├── export.py     # flatten the tree ensemble into Go-consumable JSON
└── cli.py        # `generate` and `train` subcommands
```

## Quickstart

```bash
python3 -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"

ads-ml generate            # writes data/impressions.parquet
ads-ml train               # writes artifacts/{model.json,feature_spec.json,metrics.json,...}

pytest -q                  # run tests (includes export↔LightGBM parity check)
ruff check . && ruff format --check .
```

From the repository root, `make ml-setup && make ml-pipeline` does the same.

## Artifacts

| File                  | Consumer | Purpose                                                  |
| --------------------- | -------- | -------------------------------------------------------- |
| `model.json`          | Go       | Flattened GBDT ensemble + `base_score`; scored natively. |
| `feature_spec.json`   | Go       | Feature order + categorical encodings (the contract).    |
| `metrics.json`        | humans   | AUC, log loss, precision, recall on the held-out split.  |
| `model.txt`           | Python   | Native LightGBM model for debugging / reuse.             |
| `parity_samples.json` | Go test  | Feature vectors + expected probabilities for parity.     |

## Why ordinal-encoded categoricals?

Categorical features are encoded to integer codes against fixed vocabularies in
`schema.py`. This keeps every tree split a simple `feature <= threshold` test, so
the Go scorer reproduces LightGBM's predictions exactly (see the parity test).
Unknown categories map to each vocabulary's final fallback bucket.
