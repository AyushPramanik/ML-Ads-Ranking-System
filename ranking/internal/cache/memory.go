package cache

import (
	"context"
	"sync"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

type entry struct {
	ad        store.Ad
	expiresAt time.Time
}

// MemoryCache is a concurrency-safe in-process cache with per-entry TTL, used
// when Redis is not configured and in tests.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[int64]entry
	ttl   time.Duration
	now   func() time.Time
}

// NewMemoryCache creates an in-process cache with the given TTL.
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		items: make(map[int64]entry),
		ttl:   ttl,
		now:   time.Now,
	}
}

func (c *MemoryCache) GetAd(_ context.Context, id int64) (store.Ad, bool, error) {
	c.mu.RLock()
	e, ok := c.items[id]
	c.mu.RUnlock()
	if !ok || c.now().After(e.expiresAt) {
		return store.Ad{}, false, nil
	}
	return e.ad, true, nil
}

func (c *MemoryCache) SetAd(_ context.Context, ad store.Ad) error {
	c.mu.Lock()
	c.items[ad.ID] = entry{ad: ad, expiresAt: c.now().Add(c.ttl)}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Close() error { return nil }
