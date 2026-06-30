package store

import "context"

// MemoryStore is an in-memory ad catalog used when no database is configured and
// in tests. It is safe for concurrent reads (the catalog is immutable after
// construction).
type MemoryStore struct {
	ads  []Ad
	byID map[int64]Ad
}

// NewMemoryStore builds a store from the given ads.
func NewMemoryStore(ads []Ad) *MemoryStore {
	byID := make(map[int64]Ad, len(ads))
	for _, ad := range ads {
		byID[ad.ID] = ad
	}
	return &MemoryStore{ads: ads, byID: byID}
}

// NewSeededMemoryStore builds a store pre-populated with a representative sample
// catalog, mirroring the rows seeded into PostgreSQL.
func NewSeededMemoryStore() *MemoryStore {
	return NewMemoryStore(SampleAds())
}

func (s *MemoryStore) Ads(_ context.Context) ([]Ad, error) {
	out := make([]Ad, 0, len(s.ads))
	for _, ad := range s.ads {
		if ad.CampaignStatus == "active" {
			out = append(out, ad)
		}
	}
	return out, nil
}

func (s *MemoryStore) AdsByIDs(_ context.Context, ids []int64) ([]Ad, error) {
	out := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if ad, ok := s.byID[id]; ok {
			out = append(out, ad)
		}
	}
	return out, nil
}

func (s *MemoryStore) Close() error { return nil }
