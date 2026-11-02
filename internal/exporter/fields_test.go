package exporter

import "testing"

func TestFlattenFloats(t *testing.T) {
	got, ok := flattenFloats([]float32{1, 2.5})
	if !ok || len(got) != 2 || got[0] != 1 || got[1] != 2.5 {
		t.Fatalf("flattenFloats() = %v, %v", got, ok)
	}
}

func TestMeanAndLast(t *testing.T) {
	xs := []float64{0, 1, 0, 1}
	got, ok := mean(xs)
	if !ok || got != 0.5 {
		t.Fatalf("mean() = %v, %v", got, ok)
	}
	lastGot, ok := last(xs)
	if !ok || lastGot != 1 {
		t.Fatalf("last() = %v, %v", lastGot, ok)
	}
	if _, ok := mean(nil); ok {
		t.Fatal("mean of empty slice should be absent")
	}
}

func TestStateNumber(t *testing.T) {
	cases := map[string]float64{
		"CONNECTED":   1,
		"SKY_SEARCH":  2,
		"NO_SCHEDULE": 2,
		"BOOTING":     3,
		"OFF":         5,
		"":            0,
		"OBSTRUCTED":  4,
		"NO_PINGS":    4,
	}
	for state, want := range cases {
		if got := stateNumber(state); got != want {
			t.Fatalf("stateNumber(%q) = %v, want %v", state, got, want)
		}
	}
}

func TestContainsWedge(t *testing.T) {
	if !containsWedge("wedge_fraction_obstructed") || containsWedge("fraction_obstructed") {
		t.Fatal("wedge name detection failed")
	}
}
