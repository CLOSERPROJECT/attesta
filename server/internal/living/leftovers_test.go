package living

import (
	"strings"
	"testing"
)

func TestPalletWeightLeftoverIDsSkipsWeighted(t *testing.T) {
	domain := EmptyDomain()
	domain.Pallet = EntityMap{
		"load-1.1": map[string]any{"loadId": "load-1"},
		"load-1.2": map[string]any{"loadId": "load-1", "weight": 11.0},
		"load-2.1": map[string]any{"loadId": "load-2"},
	}
	got := palletWeightLeftoverIDs(domain, "load-1")
	if len(got) != 1 || got[0] != "load-1.1" {
		t.Fatalf("got = %#v", got)
	}
}

func TestEnforceAllCompletenessPartial(t *testing.T) {
	facts := map[string]WeighRowFact{
		"a": {Weight: 1, TareKind: "euro", Idoneita: "NIR", PanelCount: 1},
	}
	msg := enforceAllCompleteness(facts, []string{"a", "b"})
	if msg == "" || !strings.Contains(msg, "b") {
		t.Fatalf("msg = %q", msg)
	}
	if enforceAllCompleteness(facts, []string{"a"}) != "" {
		t.Fatal("complete map must pass")
	}
}

func TestFilterFilledObjectWeightsRequiresRowFields(t *testing.T) {
	_, err := filterFilledObjectWeights(map[string]any{
		"p1": map[string]any{"weight": 11.0, "tareKind": "euro", "panelCount": 1.0},
	})
	if err == nil || !strings.Contains(err.Error(), "idoneita") {
		t.Fatalf("err = %v", err)
	}
	filled, err := filterFilledObjectWeights(map[string]any{
		"p1": map[string]any{"weight": 11.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0},
		"p2": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filled) != 1 {
		t.Fatalf("filled = %#v", filled)
	}
}
