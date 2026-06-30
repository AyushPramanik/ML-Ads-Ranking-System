// Package features builds model input vectors from raw request data, using the
// feature specification exported by the Python training pipeline.
//
// The spec defines the feature ordering and the categorical vocabularies. By
// loading the exact same spec the trainer produced, this package reproduces the
// Python feature engineering — including ordinal encoding and unknown-category
// fallback — so the served features match those the model was trained on.
package features

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Spec is the deserialised feature contract shared with the training pipeline.
type Spec struct {
	Version           int                 `json:"version"`
	FeatureColumns    []string            `json:"feature_columns"`
	CategoricalVocabs map[string][]string `json:"categorical_vocabs"`

	// Derived lookup tables: vocab key -> (label -> code).
	codes    map[string]map[string]int
	fallback map[string]int
}

// Input is the raw, decoded feature request for a single (user, ad) pair.
type Input struct {
	Age                    float64
	Gender                 string
	Country                string
	Device                 string
	Hour                   int
	DayOfWeek              int
	UserHistoricalCTR      float64
	AdHistoricalCTR        float64
	CampaignBudget         float64
	AdCategory             string
	UserAdPriorImpressions float64
	UserAdPriorClicks      float64
	Position               float64
}

// Load reads and indexes a feature spec from a JSON file.
func Load(path string) (*Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read feature spec: %w", err)
	}
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode feature spec: %w", err)
	}
	if len(s.FeatureColumns) == 0 {
		return nil, fmt.Errorf("feature spec has no feature columns")
	}
	s.index()
	return &s, nil
}

func (s *Spec) index() {
	s.codes = make(map[string]map[string]int, len(s.CategoricalVocabs))
	s.fallback = make(map[string]int, len(s.CategoricalVocabs))
	for key, vocab := range s.CategoricalVocabs {
		lookup := make(map[string]int, len(vocab))
		for code, label := range vocab {
			lookup[label] = code
		}
		s.codes[key] = lookup
		s.fallback[key] = len(vocab) - 1 // last bucket is the unknown fallback
	}
}

// encode maps a categorical label to its integer code; unknown labels fall back
// to the vocabulary's final bucket, matching the Python encoder.
func (s *Spec) encode(vocabKey, label string) float64 {
	if code, ok := s.codes[vocabKey][label]; ok {
		return float64(code)
	}
	return float64(s.fallback[vocabKey])
}

// Vector builds the ordered feature vector for an input, following the spec's
// column order so the result aligns with the model's expected layout.
func (s *Spec) Vector(in Input) ([]float64, error) {
	values := map[string]float64{
		"age":                       in.Age,
		"gender_code":               s.encode("gender", in.Gender),
		"country_code":              s.encode("country", in.Country),
		"device_code":               s.encode("device", in.Device),
		"hour":                      float64(in.Hour),
		"day_of_week":               float64(in.DayOfWeek),
		"user_historical_ctr":       in.UserHistoricalCTR,
		"ad_historical_ctr":         in.AdHistoricalCTR,
		"campaign_budget":           in.CampaignBudget,
		"ad_category_code":          s.encode("ad_category", in.AdCategory),
		"user_ad_prior_impressions": in.UserAdPriorImpressions,
		"user_ad_prior_clicks":      in.UserAdPriorClicks,
		"position":                  in.Position,
	}

	vec := make([]float64, len(s.FeatureColumns))
	for i, col := range s.FeatureColumns {
		v, ok := values[col]
		if !ok {
			return nil, fmt.Errorf("feature spec references unknown column %q", col)
		}
		vec[i] = v
	}
	return vec, nil
}

// NamedVector returns both the ordered vector and a column->value map, useful for
// the /features endpoint and debugging.
func (s *Spec) NamedVector(in Input) ([]float64, map[string]float64, error) {
	vec, err := s.Vector(in)
	if err != nil {
		return nil, nil, err
	}
	named := make(map[string]float64, len(vec))
	for i, col := range s.FeatureColumns {
		named[col] = vec[i]
	}
	return vec, named, nil
}

// TimeFeatures derives the hour-of-day and day-of-week (0=Monday) from an
// RFC3339 timestamp. An empty timestamp uses the provided fallback (typically
// the current time), keeping this function deterministic and testable.
func TimeFeatures(timestamp string, fallback time.Time) (hour, dayOfWeek int, err error) {
	t := fallback
	if timestamp != "" {
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid timestamp %q: %w", timestamp, err)
		}
	}
	t = t.UTC()
	// Go's Weekday is 0=Sunday..6=Saturday; map to 0=Monday..6=Sunday to match
	// the Python convention used during training.
	dow := (int(t.Weekday()) + 6) % 7
	return t.Hour(), dow, nil
}
