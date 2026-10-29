package dashboards

import (
	"encoding/json"
	"testing"
)

func TestCreateOnDemandDashboard(t *testing.T) {
	data := CreateOnDemandDashboard()
	if data == "" {
		t.Fatal("expected a generated dashboard")
	}

	var dashboard struct {
		Title string            `json:"title"`
		Rows  []json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal([]byte(data), &dashboard); err != nil {
		t.Fatalf("generated dashboard is not valid JSON: %v", err)
	}
	if dashboard.Title != "NetObserv / On Demand" {
		t.Fatalf("unexpected dashboard title %q", dashboard.Title)
	}
	if len(dashboard.Rows) == 0 {
		t.Fatal("expected generated dashboard rows")
	}
}
