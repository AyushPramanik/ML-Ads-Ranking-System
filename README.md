# ML Ads Ranking System

A production-style ad ranking system in the shape of those used by large
advertising platforms. A **Python** pipeline trains a click-through-rate (CTR)
model; a high-performance **Go** service serves ranking predictions over REST,
scoring the model natively with no Python at request time.

```
Client ──▶ Go Ranking API ──▶ Redis feature cache ──▶ PostgreSQL metadata
                  ▲
                  └── model.json + feature_spec.json ◀── Python ML training pipeline
```

See [`docs/architecture.md`](docs/architecture.md) for diagrams and design
rationale, and [`docs/api.md`](docs/api.md) for the full API reference.

## Highlights

- **Two planes, one contract.** Python trains LightGBM and exports a portable
  tree ensemble (`model.json`) plus a feature spec (`feature_spec.json`). Go
  loads both and reproduces predictions **exactly** — verified to 1e-9 by a
  cross-language parity test.
- **Native, allocation-free scoring.** The Go scorer walks the gradient-boosted
  trees directly: ~356 ns/op with 0 allocations per prediction.
- **Runs with zero infrastructure.** Postgres and Redis sit behind interfaces;
  with neither configured the service uses a seeded in-memory catalog and an
  in-process cache, so `go run` just works.
- **Realistic data.** A synthetic generator simulates users, campaigns, ads, and
  impressions with a latent click model, yielding a learnable AUC ≈ 0.74.

## Repository layout

```
ml/                     # Python CTR training pipeline (see ml/README.md)
├── src/ads_ml/         #   data gen, features, training, evaluation, export
├── scripts/            #   Go test-fixture exporter
└── tests/              #   unit + export-parity tests
ranking/                # Go ranking service (see ranking/README.md)
├── cmd/server/         #   entrypoint + graceful shutdown
└── internal/           #   config, logging, model, features, store, cache, ranking, api
deploy/postgres/        # schema + seed (init.sql)
docs/                   # architecture.md, api.md
docker-compose.yml      # postgres + redis + trainer + ranking
Makefile                # developer task runner
```

## Quickstart

### Option A — full stack with Docker (recommended)

```bash
cp .env.example .env
make up          # builds images, trains the model, serves the API

curl -s localhost:8080/health
curl -s -X POST localhost:8080/rank -H 'Content-Type: application/json' -d '{
  "user": {"age": 28, "gender": "female", "country": "US", "device": "mobile", "user_historical_ctr": 0.15},
  "top_k": 3
}'

make down        # stop and clean up
```

The `trainer` service generates the dataset and trains the model into a shared
volume; the `ranking` service starts once training completes and Postgres/Redis
are healthy.

### Option B — local, no containers

```bash
# 1. Train the model (Python)
make ml-setup            # create venv + install
make ml-pipeline         # generate dataset + train -> artifacts/

# 2. Serve (Go) — uses in-memory catalog + cache, reads ../artifacts by default
make go-run

# 3. Call it
curl -s localhost:8080/ads
```

## Common tasks

Run `make help` for the full list.

| Command         | Description                                       |
| --------------- | ------------------------------------------------- |
| `make ml-pipeline` | Generate dataset + train + export artifacts    |
| `make go-run`   | Run the ranking server locally                    |
| `make test`     | Run all Python and Go tests                       |
| `make go-bench` | Run the Go scoring/ranking benchmarks             |
| `make lint`     | Lint and format-check both services               |
| `make up` / `make down` | Start / stop the Docker stack             |

## Testing

```bash
make test        # python (pytest) + go (go test -race)
```

The suite includes the **export parity test** (`ranking/internal/model`), which
loads a committed model fixture and asserts the native Go scorer reproduces
LightGBM's probabilities — the guarantee that training and serving agree.

## Benchmarks & performance notes

Measured on an Apple M2 (`make go-bench`):

| Benchmark                | Result                                  |
| ------------------------ | --------------------------------------- |
| Single prediction        | ~356 ns/op, 0 B/op, 0 allocs (40 trees) |
| Rank full catalog (18 ads) | ~18.6 µs/op, ~14 KB, 81 allocs        |

Notes and considerations:

- Scoring cost scales linearly with `trees × candidates`. The production model
  (300 trees) predicts a single ad in low single-digit microseconds.
- Ad features are cached in Redis (cache-aside) so ranking does not hit Postgres
  on the hot path once warm.
- The scorer is read-only and stateless, so the service scales horizontally;
  reload involves swapping `model.json` and restarting.
- Feature building uses a small per-request map; for very large candidate sets a
  slice-indexed builder would cut the allocation count further.

## Configuration

All settings are environment variables (see [`.env.example`](.env.example)).
Key ones: `RANKING_HTTP_ADDR`, `RANKING_MODEL_PATH`, `RANKING_FEATURE_SPEC_PATH`,
`RANKING_POSTGRES_DSN` (empty → in-memory), `RANKING_REDIS_ADDR` (empty →
in-process), `RANKING_LOG_LEVEL`, `RANKING_LOG_FORMAT`.
