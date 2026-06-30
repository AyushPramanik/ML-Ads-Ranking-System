# API Reference

Base URL (local): `http://localhost:8080`

All request and response bodies are JSON. Errors return an appropriate HTTP
status with a body of the form:

```json
{ "error": "human-readable message" }
```

Every response carries an `X-Request-ID` header (echoed from the request if
provided), which also appears in the structured access logs.

---

## GET /health

Liveness probe.

**Response 200**

```json
{ "status": "ok", "uptime_seconds": 42.13 }
```

```bash
curl -s localhost:8080/health
```

---

## GET /ads

Returns the active ad catalog (ads belonging to active campaigns).

**Response 200**

```json
{
  "ads": [
    {
      "ad_id": 1,
      "campaign_id": 101,
      "title": "Summer Sneaker Sale",
      "category": "retail",
      "historical_ctr": 0.082,
      "campaign_budget": 5000,
      "campaign_status": "active"
    }
  ],
  "count": 1
}
```

```bash
curl -s localhost:8080/ads
```

---

## POST /features

Generates the model feature vector for a single (user, ad) context. Useful for
debugging and for verifying parity with the training-time feature pipeline.

**Request body**

| Field                        | Type   | Notes                                            |
| ---------------------------- | ------ | ------------------------------------------------ |
| `age`                        | number | User age                                         |
| `gender`                     | string | `female` \| `male` \| `unknown`                  |
| `country`                    | string | ISO-like code; unknown → fallback bucket         |
| `device`                     | string | `mobile` \| `desktop` \| `tablet`                |
| `timestamp`                  | string | RFC3339; omitted → server time. Derives hour/dow |
| `user_historical_ctr`        | number | User's historical CTR                            |
| `ad_historical_ctr`          | number | Ad's historical CTR                              |
| `campaign_budget`            | number | Campaign daily budget                            |
| `ad_category`                | string | One of the ad categories                         |
| `user_ad_prior_impressions`  | number | Prior impressions for this (user, ad)            |
| `user_ad_prior_clicks`       | number | Prior clicks for this (user, ad)                 |
| `position`                   | number | Ad slot position                                 |

**Response 200**

```json
{
  "feature_names": ["age", "gender_code", "country_code", "device_code", "hour", "day_of_week", "user_historical_ctr", "ad_historical_ctr", "campaign_budget", "ad_category_code", "user_ad_prior_impressions", "user_ad_prior_clicks", "position"],
  "features": [28, 0, 0, 0, 20, 1, 0.15, 0.12, 8000, 2, 0, 0, 0],
  "feature_map": { "age": 28, "hour": 20, "ad_category_code": 2 }
}
```

```bash
curl -s -X POST localhost:8080/features -H 'Content-Type: application/json' -d '{
  "age": 28, "gender": "female", "country": "US", "device": "mobile",
  "timestamp": "2026-06-30T20:00:00Z",
  "user_historical_ctr": 0.15, "ad_historical_ctr": 0.12,
  "campaign_budget": 8000, "ad_category": "gaming"
}'
```

---

## POST /rank

Scores and ranks candidate ads by predicted CTR.

**Request body**

| Field         | Type   | Notes                                                       |
| ------------- | ------ | ----------------------------------------------------------- |
| `user`        | object | User/context features (see below)                           |
| `candidates`  | array  | Ads to rank. **Empty/omitted → rank the entire catalog.**   |
| `position`    | number | Slot position applied to all candidates (default 0)         |
| `timestamp`   | string | RFC3339; omitted → server time                              |
| `top_k`       | number | Return only the top K results (0 → all)                     |

`user` object:

| Field                 | Type   |
| --------------------- | ------ |
| `user_id`             | number |
| `age`                 | number |
| `gender`              | string |
| `country`             | string |
| `device`              | string |
| `user_historical_ctr` | number |

`candidates[]` object:

| Field               | Type   | Notes                               |
| ------------------- | ------ | ----------------------------------- |
| `ad_id`             | number | Must exist in the catalog           |
| `prior_impressions` | number | Prior impressions for this (user, ad) |
| `prior_clicks`      | number | Prior clicks for this (user, ad)    |

**Response 200**

```json
{
  "results": [
    {
      "ad_id": 5,
      "title": "Raid Boss: Play Free",
      "category": "gaming",
      "campaign_id": 103,
      "predicted_ctr": 0.242,
      "rank": 1
    }
  ],
  "count": 1
}
```

Rank the whole catalog, top 3:

```bash
curl -s -X POST localhost:8080/rank -H 'Content-Type: application/json' -d '{
  "user": {"age": 28, "gender": "female", "country": "US", "device": "mobile", "user_historical_ctr": 0.15},
  "top_k": 3
}'
```

Rank an explicit candidate set with interaction history:

```bash
curl -s -X POST localhost:8080/rank -H 'Content-Type: application/json' -d '{
  "user": {"age": 41, "gender": "male", "country": "GB", "device": "desktop", "user_historical_ctr": 0.05},
  "candidates": [
    {"ad_id": 3, "prior_impressions": 5, "prior_clicks": 0},
    {"ad_id": 5},
    {"ad_id": 13}
  ]
}'
```

---

## Error responses

| Status | When                                                                 |
| ------ | -------------------------------------------------------------------- |
| 400    | Malformed JSON, empty body, unknown fields, or invalid `timestamp`   |
| 405    | Wrong HTTP method for a known path                                   |
| 404    | Unknown path                                                         |
| 500    | Internal error (model/feature/store failure)                        |

Request bodies are limited to 1 MiB and must contain a single JSON object;
unknown fields are rejected to catch client mistakes early.
