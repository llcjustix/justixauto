package handler

import (
	"encoding/json"
	"testing"

	"justixauto/internal/modules/retail/model"
)

func TestDemoScoringDTOProjectsMandatoryCurrentness(t *testing.T) {
	if toDemoScoring(nil, false) != nil {
		t.Fatal("missing run must remain absent")
	}
	stale, err := json.Marshal(toDemoScoring(&model.DemoScoring{ID: "run"}, false))
	if err != nil {
		t.Fatal(err)
	}
	var staleJSON map[string]any
	if err := json.Unmarshal(stale, &staleJSON); err != nil {
		t.Fatal(err)
	}
	if got, ok := staleJSON["current"].(bool); !ok || got {
		t.Fatalf("stale current = %#v", staleJSON["current"])
	}

	current := toDemoScoring(&model.DemoScoring{ID: "run"}, true)
	if current == nil || !current.Current {
		t.Fatalf("current run = %#v", current)
	}
}
