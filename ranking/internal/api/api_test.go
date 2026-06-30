package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/cache"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/features"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/model"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/ranking"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	m, err := model.Load(filepath.Join("..", "model", "testdata", "model.json"))
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	spec, err := features.Load(filepath.Join("..", "features", "testdata", "feature_spec.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	st := store.NewSeededMemoryStore()
	ranker := ranking.New(m, spec, st, cache.NewMemoryCache(time.Minute))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(NewHandlers(ranker, st, spec, logger), logger)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(t, newTestRouter(t), http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestAds(t *testing.T) {
	rec := do(t, newTestRouter(t), http.MethodGet, "/ads", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp adsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count == 0 || resp.Count != len(resp.Ads) {
		t.Errorf("count %d, ads %d", resp.Count, len(resp.Ads))
	}
}

func TestRankEndpoint(t *testing.T) {
	body := `{"user":{"age":29,"gender":"female","country":"US","device":"mobile","user_historical_ctr":0.12},"top_k":5}`
	rec := do(t, newTestRouter(t), http.MethodPost, "/rank", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp ranking.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(resp.Results))
	}
	for i := 1; i < len(resp.Results); i++ {
		if resp.Results[i-1].PredictedCTR < resp.Results[i].PredictedCTR {
			t.Error("results not sorted by predicted CTR")
		}
	}
}

func TestFeaturesEndpoint(t *testing.T) {
	body := `{"age":30,"gender":"male","country":"GB","device":"desktop","timestamp":"2026-06-30T18:30:00Z","ad_historical_ctr":0.05,"campaign_budget":1000,"ad_category":"gaming"}`
	rec := do(t, newTestRouter(t), http.MethodPost, "/features", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp featureResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Features) != len(resp.FeatureNames) {
		t.Errorf("features %d != names %d", len(resp.Features), len(resp.FeatureNames))
	}
	if resp.FeatureMap["hour"] != 18 {
		t.Errorf("hour = %v, want 18", resp.FeatureMap["hour"])
	}
}

func TestRankRejectsBadInput(t *testing.T) {
	h := newTestRouter(t)
	cases := map[string]struct {
		body string
		code int
	}{
		"malformed json": {`{not json`, http.StatusBadRequest},
		"empty body":     {``, http.StatusBadRequest},
		"unknown field":  {`{"surprise":1}`, http.StatusBadRequest},
		"bad timestamp":  {`{"user":{"age":1},"timestamp":"nope"}`, http.StatusBadRequest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/rank", tc.body)
			if rec.Code != tc.code {
				t.Errorf("status = %d, want %d (body=%s)", rec.Code, tc.code, rec.Body.String())
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := do(t, newTestRouter(t), http.MethodGet, "/rank", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
