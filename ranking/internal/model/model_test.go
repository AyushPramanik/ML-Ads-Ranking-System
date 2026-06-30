package model

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func loadFixtureModel(t *testing.T) *Model {
	t.Helper()
	m, err := Load(filepath.Join("testdata", "model.json"))
	if err != nil {
		t.Fatalf("load fixture model: %v", err)
	}
	return m
}

// TestParityWithLightGBM is the cross-language correctness guarantee: the native
// Go scorer must reproduce LightGBM's probabilities for the exported model.
func TestParityWithLightGBM(t *testing.T) {
	m := loadFixtureModel(t)

	raw, err := os.ReadFile(filepath.Join("testdata", "parity_samples.json"))
	if err != nil {
		t.Fatalf("read parity samples: %v", err)
	}
	var fixture struct {
		Samples []struct {
			Features    []float64 `json:"features"`
			Probability float64   `json:"probability"`
		} `json:"samples"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode parity samples: %v", err)
	}
	if len(fixture.Samples) == 0 {
		t.Fatal("no parity samples in fixture")
	}

	const tol = 1e-9
	maxDiff := 0.0
	for i, s := range fixture.Samples {
		got := m.Predict(s.Features)
		diff := math.Abs(got - s.Probability)
		if diff > maxDiff {
			maxDiff = diff
		}
		if diff > tol {
			t.Errorf("sample %d: got %.12f want %.12f (diff %.2e)", i, got, s.Probability, diff)
		}
	}
	t.Logf("max abs diff over %d samples: %.2e", len(fixture.Samples), maxDiff)
}

func TestLoadRejectsInvalidModels(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"bad json":          `{`,
		"unknown objective": `{"objective":"regression","num_features":1,"feature_names":["a"],"trees":[{"nodes":[{"leaf":true,"value":0.1}]}]}`,
		"feature mismatch":  `{"objective":"binary","num_features":2,"feature_names":["a"],"trees":[{"nodes":[{"leaf":true,"value":0.1}]}]}`,
		"no trees":          `{"objective":"binary","num_features":1,"feature_names":["a"],"trees":[]}`,
		"bad child index":   `{"objective":"binary","num_features":1,"feature_names":["a"],"trees":[{"nodes":[{"leaf":false,"feature":0,"threshold":0.5,"left":5,"right":1}]}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, "m.json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestRawScoreWalksThreshold(t *testing.T) {
	// A single tree: feature[0] <= 0.5 ? leaf 1.0 : leaf -1.0, base 0.
	m := &Model{
		Objective:    "binary",
		NumFeatures:  1,
		FeatureNames: []string{"x"},
		Trees: []tree{{Nodes: []node{
			{Leaf: false, Feature: 0, Threshold: 0.5, Left: 1, Right: 2},
			{Leaf: true, Value: 1.0},
			{Leaf: true, Value: -1.0},
		}}},
	}
	if got := m.RawScore([]float64{0.0}); got != 1.0 {
		t.Errorf("left branch: got %v want 1.0", got)
	}
	if got := m.RawScore([]float64{1.0}); got != -1.0 {
		t.Errorf("right branch: got %v want -1.0", got)
	}
	// NaN with default_left=false should go right.
	m.Trees[0].Nodes[0].DefaultLeft = false
	if got := m.RawScore([]float64{math.NaN()}); got != -1.0 {
		t.Errorf("nan branch: got %v want -1.0", got)
	}
}

func BenchmarkPredict(b *testing.B) {
	m, err := Load(filepath.Join("testdata", "model.json"))
	if err != nil {
		b.Fatal(err)
	}
	features := make([]float64, m.NumFeatures)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Predict(features)
	}
}
