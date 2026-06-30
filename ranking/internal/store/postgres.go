package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore reads ad metadata from PostgreSQL via a connection pool.
type PostgresStore struct {
	pool *pgxpool.Pool
}

const adSelect = `
	SELECT a.id, a.campaign_id, a.title, a.category,
	       a.historical_ctr, c.budget, c.status
	FROM ads a
	JOIN campaigns c ON c.id = a.campaign_id`

// NewPostgresStore connects to PostgreSQL using the given DSN and verifies the
// connection with a ping.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Ads(ctx context.Context) ([]Ad, error) {
	rows, err := s.pool.Query(ctx, adSelect+" WHERE c.status = 'active' ORDER BY a.id")
	if err != nil {
		return nil, fmt.Errorf("query ads: %w", err)
	}
	return collectAds(rows)
}

func (s *PostgresStore) AdsByIDs(ctx context.Context, ids []int64) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, adSelect+" WHERE a.id = ANY($1)", ids)
	if err != nil {
		return nil, fmt.Errorf("query ads by id: %w", err)
	}
	return collectAds(rows)
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func collectAds(rows pgx.Rows) ([]Ad, error) {
	defer rows.Close()
	var ads []Ad
	for rows.Next() {
		var ad Ad
		if err := rows.Scan(
			&ad.ID, &ad.CampaignID, &ad.Title, &ad.Category,
			&ad.HistoricalCTR, &ad.CampaignBudget, &ad.CampaignStatus,
		); err != nil {
			return nil, fmt.Errorf("scan ad: %w", err)
		}
		ads = append(ads, ad)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ads: %w", err)
	}
	return ads, nil
}
