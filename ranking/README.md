# ranking — Go ads ranking service

High-performance REST service that scores and ranks candidate ads by predicted
click-through rate. It loads the CTR model and feature spec exported by the
Python pipeline and evaluates the gradient-boosted tree ensemble natively — no
Python at serving time.

## Layout

```
cmd/server/         # main: wiring + graceful shutdown
internal/
├── config/         # environment-driven configuration
├── logging/        # structured slog setup
├── model/          # portable GBDT model loader + native scorer
├── features/       # feature-spec loader + feature vector builder (matches Python)
├── store/          # ad metadata: PostgreSQL + in-memory implementations
├── cache/          # ad feature cache: Redis + in-process implementations
├── ranking/        # candidate assembly, scoring, and ordering
└── api/            # handlers, middleware, router, JSON helpers
```

## Design notes

- **No infra required to run.** With no `RANKING_POSTGRES_DSN` the service uses a
  seeded in-memory catalog; with no `RANKING_REDIS_ADDR` it uses an in-process
  cache. Both are interface implementations swapped at startup.
- **Cache-aside.** Explicit candidate ads are resolved through the cache first,
  falling back to the store on a miss and populating the cache.
- **Exact parity with LightGBM.** The native scorer reproduces LightGBM's
  predictions (see `internal/model` parity test) because every split is a numeric
  threshold test.
- **Middleware stack:** request ID → structured request logging → panic recovery.

## Develop

```bash
go build ./cmd/server      # or: make go-build (from repo root)
go test -race ./...        # run tests
go test -bench=. -benchmem ./internal/...   # benchmarks
go vet ./... && gofmt -l . # lint/format checks
```

The service expects `model.json` and `feature_spec.json` (from the Python
pipeline) at the paths in `RANKING_MODEL_PATH` / `RANKING_FEATURE_SPEC_PATH`.
Defaults point at `../artifacts/` for local runs after `make ml-pipeline`.

## Endpoints

See [`docs/api.md`](../docs/api.md) for full request/response schemas.

| Method | Path        | Purpose                                   |
| ------ | ----------- | ----------------------------------------- |
| GET    | `/health`   | Liveness check + uptime                   |
| GET    | `/ads`      | Active ad catalog                         |
| POST   | `/rank`     | Rank candidate ads by predicted CTR       |
| POST   | `/features` | Generate the model feature vector for input |
