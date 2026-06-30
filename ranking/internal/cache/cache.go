// Package cache provides a cache-aside layer over ad metadata.
//
// In production the cache is backed by Redis; with no Redis configured it falls
// back to an in-process cache. Ad records are cached because their ad-side
// features (historical CTR, campaign budget, category) are read on every ranking
// request but change infrequently.
package cache

import (
	"context"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

// Cache stores ad records keyed by ad ID with a TTL.
type Cache interface {
	// GetAd returns the cached ad and whether it was present.
	GetAd(ctx context.Context, id int64) (store.Ad, bool, error)
	// SetAd caches an ad record.
	SetAd(ctx context.Context, ad store.Ad) error
	// Close releases any underlying resources.
	Close() error
}
