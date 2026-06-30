package features

import (
	"path/filepath"
	"testing"
	"time"
)

func loadFixtureSpec(t *testing.T) *Spec {
	t.Helper()
	s, err := Load(filepath.Join("testdata", "feature_spec.json"))
	if err != nil {
		t.Fatalf("load fixture spec: %v", err)
	}
	return s
}

func TestVectorOrderMatchesSpec(t *testing.T) {
	s := loadFixtureSpec(t)
	in := Input{
		Age: 30, Gender: "male", Country: "US", Device: "mobile",
		Hour: 18, DayOfWeek: 5, UserHistoricalCTR: 0.1, AdHistoricalCTR: 0.05,
		CampaignBudget: 1000, AdCategory: "retail",
		UserAdPriorImpressions: 3, UserAdPriorClicks: 1, Position: 2,
	}
	vec, named, err := s.NamedVector(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != len(s.FeatureColumns) {
		t.Fatalf("vector length %d != columns %d", len(vec), len(s.FeatureColumns))
	}
	for i, col := range s.FeatureColumns {
		if vec[i] != named[col] {
			t.Errorf("column %q: vec[%d]=%v but map=%v", col, i, vec[i], named[col])
		}
	}
	// gender "male" is index 1; device "mobile" index 0; category "retail" index 0.
	if named["gender_code"] != 1 {
		t.Errorf("gender_code = %v, want 1", named["gender_code"])
	}
	if named["age"] != 30 {
		t.Errorf("age = %v, want 30", named["age"])
	}
}

func TestUnknownCategoryFallsBackToLastBucket(t *testing.T) {
	s := loadFixtureSpec(t)
	vocab := s.CategoricalVocabs["country"]
	fallback := float64(len(vocab) - 1)
	if got := s.encode("country", "ATLANTIS"); got != fallback {
		t.Errorf("unknown country code = %v, want %v", got, fallback)
	}
	if got := s.encode("country", "US"); got != 0 {
		t.Errorf("US code = %v, want 0", got)
	}
}

func TestTimeFeatures(t *testing.T) {
	// 2026-06-30 is a Tuesday (Monday=0 => 1), 18:30 UTC => hour 18.
	hour, dow, err := TimeFeatures("2026-06-30T18:30:00Z", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if hour != 18 {
		t.Errorf("hour = %d, want 18", hour)
	}
	if dow != 1 {
		t.Errorf("day_of_week = %d, want 1 (Tuesday)", dow)
	}

	// Saturday should map to 5 (weekend bucket used in training).
	_, satDow, _ := TimeFeatures("2026-07-04T10:00:00Z", time.Time{})
	if satDow != 5 {
		t.Errorf("Saturday day_of_week = %d, want 5", satDow)
	}
}

func TestTimeFeaturesUsesFallbackWhenEmpty(t *testing.T) {
	fallback := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC) // Thursday
	hour, dow, err := TimeFeatures("", fallback)
	if err != nil {
		t.Fatal(err)
	}
	if hour != 9 || dow != 3 {
		t.Errorf("got hour=%d dow=%d, want 9 and 3", hour, dow)
	}
}

func TestTimeFeaturesRejectsBadTimestamp(t *testing.T) {
	if _, _, err := TimeFeatures("not-a-time", time.Now()); err == nil {
		t.Fatal("expected error for malformed timestamp")
	}
}
