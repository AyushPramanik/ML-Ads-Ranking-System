// Package ranking scores and orders candidate ads by predicted click-through
// rate. It is the heart of the service: it assembles features for each candidate
// (user context + cached ad-side features + interaction history + time), scores
// them with the CTR model, and returns ads sorted by predicted CTR.
package ranking

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/cache"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/features"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/model"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

// UserContext describes the user and request context for a ranking call.
type UserContext struct {
	UserID            int64   `json:"user_id"`
	Age               float64 `json:"age"`
	Gender            string  `json:"gender"`
	Country           string  `json:"country"`
	Device            string  `json:"device"`
	UserHistoricalCTR float64 `json:"user_historical_ctr"`
}

// Candidate is an ad to rank, optionally carrying per-(user,ad) interaction
// history. When the candidate list is empty the entire active catalog is ranked.
type Candidate struct {
	AdID             int64   `json:"ad_id"`
	PriorImpressions float64 `json:"prior_impressions"`
	PriorClicks      float64 `json:"prior_clicks"`
}

// Request is a ranking request.
type Request struct {
	User       UserContext `json:"user"`
	Candidates []Candidate `json:"candidates"`
	Position   float64     `json:"position"`
	Timestamp  string      `json:"timestamp"` // RFC3339; empty means "now"
	TopK       int         `json:"top_k"`     // 0 means "return all"
}

// RankedAd is a single scored, ranked result.
type RankedAd struct {
	AdID         int64   `json:"ad_id"`
	Title        string  `json:"title"`
	Category     string  `json:"category"`
	CampaignID   int64   `json:"campaign_id"`
	PredictedCTR float64 `json:"predicted_ctr"`
	Rank         int     `json:"rank"`
}

// Response is the ranking result.
type Response struct {
	Results []RankedAd `json:"results"`
	Count   int        `json:"count"`
}

// Ranker holds the dependencies needed to rank ads.
type Ranker struct {
	model *model.Model
	spec  *features.Spec
	store store.Store
	cache cache.Cache
	now   func() time.Time
}

// New constructs a Ranker.
func New(m *model.Model, spec *features.Spec, st store.Store, c cache.Cache) *Ranker {
	return &Ranker{model: m, spec: spec, store: st, cache: c, now: time.Now}
}

// Rank scores the candidate ads and returns them ordered by predicted CTR.
func (r *Ranker) Rank(ctx context.Context, req Request) (Response, error) {
	hour, dayOfWeek, err := features.TimeFeatures(req.Timestamp, r.now())
	if err != nil {
		return Response{}, err
	}

	ads, interactions, err := r.resolveCandidates(ctx, req.Candidates)
	if err != nil {
		return Response{}, err
	}

	results := make([]RankedAd, 0, len(ads))
	for _, ad := range ads {
		hist := interactions[ad.ID]
		in := features.Input{
			Age:                    req.User.Age,
			Gender:                 req.User.Gender,
			Country:                req.User.Country,
			Device:                 req.User.Device,
			Hour:                   hour,
			DayOfWeek:              dayOfWeek,
			UserHistoricalCTR:      req.User.UserHistoricalCTR,
			AdHistoricalCTR:        ad.HistoricalCTR,
			CampaignBudget:         ad.CampaignBudget,
			AdCategory:             ad.Category,
			UserAdPriorImpressions: hist.PriorImpressions,
			UserAdPriorClicks:      hist.PriorClicks,
			Position:               req.Position,
		}
		vec, err := r.spec.Vector(in)
		if err != nil {
			return Response{}, fmt.Errorf("build features for ad %d: %w", ad.ID, err)
		}
		results = append(results, RankedAd{
			AdID:         ad.ID,
			Title:        ad.Title,
			Category:     ad.Category,
			CampaignID:   ad.CampaignID,
			PredictedCTR: r.model.Predict(vec),
		})
	}

	sortByCTR(results)
	if req.TopK > 0 && req.TopK < len(results) {
		results = results[:req.TopK]
	}
	for i := range results {
		results[i].Rank = i + 1
	}

	return Response{Results: results, Count: len(results)}, nil
}

// resolveCandidates returns the ads to score and a map of interaction history by
// ad ID. With no explicit candidates it ranks the full active catalog; otherwise
// it resolves each ad through the cache, falling back to the store on a miss.
func (r *Ranker) resolveCandidates(
	ctx context.Context, candidates []Candidate,
) ([]store.Ad, map[int64]Candidate, error) {
	if len(candidates) == 0 {
		ads, err := r.store.Ads(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("load catalog: %w", err)
		}
		return ads, nil, nil
	}

	interactions := make(map[int64]Candidate, len(candidates))
	ids := make([]int64, 0, len(candidates))
	for _, c := range candidates {
		interactions[c.AdID] = c
		ids = append(ids, c.AdID)
	}

	ads := make([]store.Ad, 0, len(ids))
	misses := make([]int64, 0)
	for _, id := range ids {
		if ad, ok, err := r.cache.GetAd(ctx, id); err != nil {
			return nil, nil, err
		} else if ok {
			ads = append(ads, ad)
		} else {
			misses = append(misses, id)
		}
	}

	if len(misses) > 0 {
		fetched, err := r.store.AdsByIDs(ctx, misses)
		if err != nil {
			return nil, nil, fmt.Errorf("load ads: %w", err)
		}
		for _, ad := range fetched {
			if err := r.cache.SetAd(ctx, ad); err != nil {
				return nil, nil, err
			}
			ads = append(ads, ad)
		}
	}

	return ads, interactions, nil
}

// sortByCTR orders results by descending predicted CTR, breaking ties by ad ID
// for deterministic output.
func sortByCTR(results []RankedAd) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].PredictedCTR != results[j].PredictedCTR {
			return results[i].PredictedCTR > results[j].PredictedCTR
		}
		return results[i].AdID < results[j].AdID
	})
}
