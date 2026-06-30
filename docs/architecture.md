# Architecture

The system splits cleanly into an **offline training plane** (Python) and an
**online serving plane** (Go), connected by two serialized artifacts: the model
and the feature spec.

## Component overview

```mermaid
flowchart TD
    Client[Client] -->|REST| API[Go Ranking API]

    subgraph Serving["Online serving plane (Go)"]
        API --> MW[Middleware: request-id, logging, recovery]
        MW --> RANK[Ranker]
        RANK --> SCORER[Native GBDT scorer]
        RANK --> FEAT[Feature builder]
        RANK -->|cache-aside| CACHE[(Redis feature cache)]
        RANK -->|catalog| STORE[(PostgreSQL ad metadata)]
    end

    subgraph Training["Offline training plane (Python)"]
        GEN[Synthetic data generator] --> DS[(impressions.parquet)]
        DS --> TRAIN[LightGBM trainer + evaluator]
        TRAIN --> EXPORT[Model exporter]
    end

    EXPORT -->|model.json| SCORER
    EXPORT -->|feature_spec.json| FEAT
    CACHE -. miss .-> STORE
```

## Why this split

| Concern            | Plane    | Rationale                                                            |
| ------------------ | -------- | -------------------------------------------------------------------- |
| Model training     | Python   | Mature ML ecosystem (LightGBM, scikit-learn, pandas).                |
| Online inference   | Go       | Low-latency, low-allocation scoring; easy concurrency and deploys.   |
| Model transport    | JSON     | Language-agnostic; Go scores the tree ensemble with no Python.       |
| Feature contract   | JSON     | One spec, produced by Python, consumed by Go — no drift.             |

## The feature contract

Feature ordering and categorical vocabularies live in exactly one place
(`ml/src/ads_ml/schema.py`). Training serialises them to `feature_spec.json`; the
Go service loads that file and reproduces the encoding — including ordinal codes
and unknown-category fallback — so served features match trained features.

Because categoricals are ordinal-encoded, every decision-tree split is a numeric
`feature <= threshold` test. This lets the Go scorer walk each tree to a leaf and
sum leaf values, reproducing LightGBM's prediction exactly (asserted to 1e-9 by
the parity test in `ranking/internal/model`).

## Request flow: POST /rank

```mermaid
sequenceDiagram
    participant C as Client
    participant A as Ranking API
    participant K as Ranker
    participant R as Redis
    participant P as Postgres
    participant M as GBDT scorer

    C->>A: POST /rank {user, candidates?, top_k}
    A->>K: Rank(request)
    alt no candidates given
        K->>P: load active catalog
    else explicit candidates
        K->>R: GetAd(id) per candidate
        R-->>K: hits
        K->>P: AdsByIDs(misses)
        K->>R: SetAd(fetched)
    end
    loop each candidate
        K->>K: build feature vector (user + ad + time + history)
        K->>M: Predict(features)
        M-->>K: predicted CTR
    end
    K-->>A: ranked ads (sorted desc, top_k)
    A-->>C: 200 {results, count}
```

## Data model

```mermaid
erDiagram
    CAMPAIGNS ||--o{ ADS : contains
    USERS ||--o{ IMPRESSIONS : generates
    ADS ||--o{ IMPRESSIONS : shown_in
    IMPRESSIONS ||--o| CLICKS : may_result_in

    CAMPAIGNS {
        bigint id PK
        text name
        text advertiser
        text category
        numeric budget
        text status
    }
    ADS {
        bigint id PK
        bigint campaign_id FK
        text title
        text category
        numeric historical_ctr
    }
    USERS {
        bigint id PK
        int age
        text gender
        text country
    }
    IMPRESSIONS {
        bigint id PK
        bigint user_id FK
        bigint ad_id FK
        text device
        int position
    }
    CLICKS {
        bigint id PK
        bigint impression_id FK
    }
```

## Deployment topology

`docker-compose` runs four services: `postgres`, `redis`, a one-shot `trainer`
that writes the model artifacts to a shared volume, and the `ranking` service
that serves the API once training completes and the datastores are healthy.

## Graceful degradation

The store and cache are interfaces. With no PostgreSQL DSN the service falls back
to a seeded in-memory catalog; with no Redis address it falls back to an
in-process cache. The service therefore runs end-to-end with zero infrastructure,
which keeps development and testing friction low.
