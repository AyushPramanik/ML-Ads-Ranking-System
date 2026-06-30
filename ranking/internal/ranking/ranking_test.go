package ranking

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/cache"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/features"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/model"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

// countingStore wraps a store and counts AdsByIDs calls to assert cache-aside.
type countingStore struct {
	store.Store
	adsByIDsCalls int
}

func (c *countingStore) AdsByIDs(ctx context.Context, ids []int64) ([]store.Ad, error) {
	c.adsByIDsCalls++
	return c.Store.AdsByIDs(ctx, ids)
}

func newTestRanker(t testing.TB, st store.Store, c cache.Cache) *Ranker {
	t.Helper()
	m, err := model.Load(filepath.Join("..", "model", "testdata", "model.json"))
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	spec, err := features.Load(filepath.Join("..", "features", "testdata", "feature_spec.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	r := New(m, spec, st, c)
	r.now = func() time.Time { return time.Date(2026, 6, 30, 18, 0, 0, 0, time.UTC) }
	return r
}

func sampleUser() UserContext {
	return UserContext{
		UserID: 1, Age: 29, Gender: "female", Country: "US",
		Device: "mobile", UserHistoricalCTR: 0.12,
	}
}

func TestRankFullCatalogIsSortedAndRanked(t *testing.T) {
	r := newTestRanker(t, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))

	resp, err := r.Rank(context.Background(), Request{User: sampleUser()})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count == 0 || resp.Count != len(resp.Results) {
		t.Fatalf("count %d does not match results %d", resp.Count, len(resp.Results))
	}
	for i := 1; i < len(resp.Results); i++ {
		if resp.Results[i-1].PredictedCTR < resp.Results[i].PredictedCTR {
			t.Errorf("results not sorted at %d: %.4f < %.4f",
				i, resp.Results[i-1].PredictedCTR, resp.Results[i].PredictedCTR)
		}
		if resp.Results[i].Rank != i+1 {
			t.Errorf("rank at %d = %d, want %d", i, resp.Results[i].Rank, i+1)
		}
	}
	for _, res := range resp.Results {
		if res.PredictedCTR < 0 || res.PredictedCTR > 1 {
			t.Errorf("predicted CTR out of [0,1]: %v", res.PredictedCTR)
		}
	}
}

func TestRankPausedCampaignExcludedFromCatalog(t *testing.T) {
	r := newTestRanker(t, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))
	resp, err := r.Rank(context.Background(), Request{User: sampleUser()})
	if err != nil {
		t.Fatal(err)
	}
	for _, res := range resp.Results {
		if res.AdID == 19 { // ad 19 belongs to a paused campaign
			t.Error("paused-campaign ad 19 should not appear in full-catalog ranking")
		}
	}
}

func TestRankTopKTruncates(t *testing.T) {
	r := newTestRanker(t, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))
	resp, err := r.Rank(context.Background(), Request{User: sampleUser(), TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("top_k=3 returned %d results", len(resp.Results))
	}
}

func TestRankExplicitCandidatesUseCacheAside(t *testing.T) {
	cs := &countingStore{Store: store.NewSeededMemoryStore()}
	c := cache.NewMemoryCache(time.Minute)
	r := newTestRanker(t, cs, c)

	req := Request{
		User: sampleUser(),
		Candidates: []Candidate{
			{AdID: 5, PriorImpressions: 4, PriorClicks: 2},
			{AdID: 7},
		},
	}

	first, err := r.Rank(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Count != 2 {
		t.Fatalf("expected 2 results, got %d", first.Count)
	}
	if cs.adsByIDsCalls != 1 {
		t.Fatalf("expected 1 store fetch on cold cache, got %d", cs.adsByIDsCalls)
	}

	// Second call should be served entirely from cache.
	if _, err := r.Rank(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if cs.adsByIDsCalls != 1 {
		t.Errorf("expected store not hit on warm cache, calls=%d", cs.adsByIDsCalls)
	}
}

func TestRankUnknownCandidateIsSkipped(t *testing.T) {
	r := newTestRanker(t, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))
	resp, err := r.Rank(context.Background(), Request{
		User:       sampleUser(),
		Candidates: []Candidate{{AdID: 5}, {AdID: 99999}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || resp.Results[0].AdID != 5 {
		t.Fatalf("expected only ad 5, got %+v", resp.Results)
	}
}

func TestRankRejectsBadTimestamp(t *testing.T) {
	r := newTestRanker(t, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))
	_, err := r.Rank(context.Background(), Request{User: sampleUser(), Timestamp: "nope"})
	if err == nil {
		t.Fatal("expected error for bad timestamp")
	}
}

func BenchmarkRankFullCatalog(b *testing.B) {
	r := newTestRanker(b, store.NewSeededMemoryStore(), cache.NewMemoryCache(time.Minute))
	req := Request{User: sampleUser()}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Rank(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
