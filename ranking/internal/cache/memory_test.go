package cache

import (
	"context"
	"testing"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

func TestMemoryCacheStoreAndGet(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	ad := store.Ad{ID: 7, Title: "Cheap Flights", Category: "travel"}
	if err := c.SetAd(context.Background(), ad); err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.GetAd(context.Background(), 7)
	if err != nil || !ok {
		t.Fatalf("expected hit, ok=%v err=%v", ok, err)
	}
	if got.Title != "Cheap Flights" {
		t.Errorf("got %q, want Cheap Flights", got.Title)
	}
}

func TestMemoryCacheMiss(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	if _, ok, _ := c.GetAd(context.Background(), 404); ok {
		t.Error("expected miss for absent key")
	}
}

func TestMemoryCacheExpiry(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	if err := c.SetAd(context.Background(), store.Ad{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.GetAd(context.Background(), 1); !ok {
		t.Fatal("expected hit before expiry")
	}

	now = now.Add(2 * time.Minute) // advance past TTL
	if _, ok, _ := c.GetAd(context.Background(), 1); ok {
		t.Error("expected miss after TTL expiry")
	}
}
