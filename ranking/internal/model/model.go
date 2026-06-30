// Package model loads the portable CTR model exported by the Python pipeline and
// scores feature vectors natively, with no Python runtime at serving time.
//
// The model is a gradient-boosted decision-tree ensemble. Each tree is a flat
// array of nodes; scoring walks every tree from its root to a leaf and sums the
// leaf values. For the binary objective the raw sum (plus a base score) is passed
// through a sigmoid to yield a click-through probability. Because all features
// are numeric, every split is a `feature <= threshold` test, so this evaluator
// reproduces LightGBM's predictions exactly (verified by the parity test).
package model

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// node is a single tree node. Internal nodes test a feature against a threshold;
// leaf nodes carry an output value.
type node struct {
	Leaf        bool    `json:"leaf"`
	Value       float64 `json:"value"`
	Feature     int     `json:"feature"`
	Threshold   float64 `json:"threshold"`
	DefaultLeft bool    `json:"default_left"`
	Left        int     `json:"left"`
	Right       int     `json:"right"`
}

type tree struct {
	Nodes []node `json:"nodes"`
}

// Model is the deserialised, validated tree ensemble.
type Model struct {
	FormatVersion int      `json:"format_version"`
	Objective     string   `json:"objective"`
	NumFeatures   int      `json:"num_features"`
	FeatureNames  []string `json:"feature_names"`
	BaseScore     float64  `json:"base_score"`
	Trees         []tree   `json:"trees"`
}

// Load reads and validates a model from a JSON file.
func Load(path string) (*Model, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read model: %w", err)
	}
	var m Model
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode model: %w", err)
	}
	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("invalid model %q: %w", path, err)
	}
	return &m, nil
}

func (m *Model) validate() error {
	if m.Objective != "binary" {
		return fmt.Errorf("unsupported objective %q", m.Objective)
	}
	if m.NumFeatures <= 0 || len(m.FeatureNames) != m.NumFeatures {
		return fmt.Errorf("num_features (%d) does not match feature_names (%d)",
			m.NumFeatures, len(m.FeatureNames))
	}
	if len(m.Trees) == 0 {
		return fmt.Errorf("model has no trees")
	}
	for ti := range m.Trees {
		nodes := m.Trees[ti].Nodes
		if len(nodes) == 0 {
			return fmt.Errorf("tree %d has no nodes", ti)
		}
		for ni := range nodes {
			n := nodes[ni]
			if n.Leaf {
				continue
			}
			if n.Feature < 0 || n.Feature >= m.NumFeatures {
				return fmt.Errorf("tree %d node %d: feature %d out of range", ti, ni, n.Feature)
			}
			if n.Left < 0 || n.Left >= len(nodes) || n.Right < 0 || n.Right >= len(nodes) {
				return fmt.Errorf("tree %d node %d: child index out of range", ti, ni)
			}
		}
	}
	return nil
}

// RawScore returns the summed leaf values (plus base score) for a feature vector.
func (m *Model) RawScore(features []float64) float64 {
	sum := m.BaseScore
	for ti := range m.Trees {
		nodes := m.Trees[ti].Nodes
		idx := 0
		for !nodes[idx].Leaf {
			n := &nodes[idx]
			v := features[n.Feature]
			switch {
			case math.IsNaN(v):
				if n.DefaultLeft {
					idx = n.Left
				} else {
					idx = n.Right
				}
			case v <= n.Threshold:
				idx = n.Left
			default:
				idx = n.Right
			}
		}
		sum += nodes[idx].Value
	}
	return sum
}

// Predict returns the click-through probability for a feature vector.
func (m *Model) Predict(features []float64) float64 {
	return sigmoid(m.RawScore(features))
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}
