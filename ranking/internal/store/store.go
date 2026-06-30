// Package store provides access to ad and campaign metadata.
//
// The Store interface is backed by PostgreSQL in production and by an in-memory
// catalog in development and tests, so the service runs with or without a
// database.
package store

import "context"

// Ad is the metadata and ad-side features for a single ad.
type Ad struct {
	ID             int64   `json:"ad_id"`
	CampaignID     int64   `json:"campaign_id"`
	Title          string  `json:"title"`
	Category       string  `json:"category"`
	HistoricalCTR  float64 `json:"historical_ctr"`
	CampaignBudget float64 `json:"campaign_budget"`
	CampaignStatus string  `json:"campaign_status"`
}

// Store is the read interface over ad metadata. Only active-campaign ads are
// returned by Ads; AdsByIDs returns whatever matches regardless of status so
// explicit candidate lists are honoured.
type Store interface {
	// Ads returns all ads belonging to active campaigns.
	Ads(ctx context.Context) ([]Ad, error)
	// AdsByIDs returns the ads matching the given IDs, in arbitrary order.
	AdsByIDs(ctx context.Context, ids []int64) ([]Ad, error)
	// Close releases any underlying resources.
	Close() error
}
