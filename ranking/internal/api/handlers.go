package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/features"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/ranking"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

// maxRequestBody bounds request bodies to guard against abuse.
const maxRequestBody = 1 << 20 // 1 MiB

// Handlers bundles the dependencies the HTTP handlers need.
type Handlers struct {
	ranker    *ranking.Ranker
	store     store.Store
	spec      *features.Spec
	logger    *slog.Logger
	startTime time.Time
	now       func() time.Time
}

// NewHandlers constructs the HTTP handlers.
func NewHandlers(
	ranker *ranking.Ranker, st store.Store, spec *features.Spec, logger *slog.Logger,
) *Handlers {
	return &Handlers{
		ranker:    ranker,
		store:     st,
		spec:      spec,
		logger:    logger,
		startTime: time.Now(),
		now:       time.Now,
	}
}

// healthResponse is returned by GET /health.
type healthResponse struct {
	Status        string  `json:"status"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

// Health is a liveness check. GET /health
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:        "ok",
		UptimeSeconds: h.now().Sub(h.startTime).Seconds(),
	})
}

// adsResponse is returned by GET /ads.
type adsResponse struct {
	Ads   []store.Ad `json:"ads"`
	Count int        `json:"count"`
}

// Ads returns the active ad catalog. GET /ads
func (h *Handlers) Ads(w http.ResponseWriter, r *http.Request) {
	ads, err := h.store.Ads(r.Context())
	if err != nil {
		h.logger.Error("list ads", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load ads")
		return
	}
	writeJSON(w, http.StatusOK, adsResponse{Ads: ads, Count: len(ads)})
}

// Rank scores and orders candidate ads. POST /rank
func (h *Handlers) Rank(w http.ResponseWriter, r *http.Request) {
	var req ranking.Request
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.ranker.Rank(r.Context(), req)
	if err != nil {
		if isClientError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("rank", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to rank ads")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// featureRequest is the body for POST /features.
type featureRequest struct {
	Age                    float64 `json:"age"`
	Gender                 string  `json:"gender"`
	Country                string  `json:"country"`
	Device                 string  `json:"device"`
	Timestamp              string  `json:"timestamp"`
	UserHistoricalCTR      float64 `json:"user_historical_ctr"`
	AdHistoricalCTR        float64 `json:"ad_historical_ctr"`
	CampaignBudget         float64 `json:"campaign_budget"`
	AdCategory             string  `json:"ad_category"`
	UserAdPriorImpressions float64 `json:"user_ad_prior_impressions"`
	UserAdPriorClicks      float64 `json:"user_ad_prior_clicks"`
	Position               float64 `json:"position"`
}

// featureResponse is returned by POST /features.
type featureResponse struct {
	FeatureNames []string           `json:"feature_names"`
	Features     []float64          `json:"features"`
	FeatureMap   map[string]float64 `json:"feature_map"`
}

// Features generates the model input vector for a raw request. POST /features
func (h *Handlers) Features(w http.ResponseWriter, r *http.Request) {
	var req featureRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hour, dayOfWeek, err := features.TimeFeatures(req.Timestamp, h.now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	in := features.Input{
		Age:                    req.Age,
		Gender:                 req.Gender,
		Country:                req.Country,
		Device:                 req.Device,
		Hour:                   hour,
		DayOfWeek:              dayOfWeek,
		UserHistoricalCTR:      req.UserHistoricalCTR,
		AdHistoricalCTR:        req.AdHistoricalCTR,
		CampaignBudget:         req.CampaignBudget,
		AdCategory:             req.AdCategory,
		UserAdPriorImpressions: req.UserAdPriorImpressions,
		UserAdPriorClicks:      req.UserAdPriorClicks,
		Position:               req.Position,
	}

	vec, named, err := h.spec.NamedVector(in)
	if err != nil {
		h.logger.Error("build features", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to build features")
		return
	}
	writeJSON(w, http.StatusOK, featureResponse{
		FeatureNames: h.spec.FeatureColumns,
		Features:     vec,
		FeatureMap:   named,
	})
}

// decodeJSON decodes a JSON request body, rejecting unknown fields and oversized
// or malformed input with a descriptive error.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return errors.New("invalid request body: " + err.Error())
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// isClientError reports whether a ranking error stems from bad input (e.g. an
// invalid timestamp) rather than a server fault.
func isClientError(err error) bool {
	var terr *time.ParseError
	return errors.As(err, &terr)
}
