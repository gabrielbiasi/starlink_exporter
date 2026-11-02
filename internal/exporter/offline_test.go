package exporter

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestEmitOffline(t *testing.T) {
	ch := make(chan prometheus.Metric, 8)
	emitOffline(ch, time.Now())
	close(ch)

	got := map[string]float64{}
	var stateLabel string
	for m := range ch {
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatal(err)
		}
		name := descName(t, m)
		if pb.Gauge == nil || pb.Gauge.Value == nil {
			t.Fatalf("%s has no gauge value", name)
		}
		got[name] = *pb.Gauge.Value
		if name == "starlink_dish_state_name" {
			if len(pb.Label) != 1 || pb.Label[0].GetName() != "state" {
				t.Fatalf("state_name labels = %v", pb.Label)
			}
			stateLabel = pb.Label[0].GetValue()
		}
	}

	if got["starlink_dish_up"] != 0 || got["starlink_dish_history_up"] != 0 {
		t.Fatalf("reachability gauges = %#v", got)
	}
	if got["starlink_dish_state"] != 5 {
		t.Fatalf("state = %v, want 5", got["starlink_dish_state"])
	}
	if stateLabel != "OFF" || got["starlink_dish_state_name"] != 1 {
		t.Fatalf("state_name = %s %v", stateLabel, got["starlink_dish_state_name"])
	}
	if _, ok := got["starlink_dish_scrape_duration_seconds"]; !ok {
		t.Fatal("missing scrape duration")
	}
}

func descName(t *testing.T, m prometheus.Metric) string {
	t.Helper()
	const key = `fqName: "`
	s := m.Desc().String()
	i := strings.Index(s, key)
	if i < 0 {
		t.Fatalf("desc %q has no fqName", s)
	}
	s = s[i+len(key):]
	j := strings.Index(s, `"`)
	if j < 0 {
		t.Fatalf("desc %q has an unterminated fqName", m.Desc().String())
	}
	return s[:j]
}
