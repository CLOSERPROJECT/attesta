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

func TestLineLoadLeftoverIDsSkipsDone(t *testing.T) {
	domain := EmptyDomain()
	domain.Pallet = EntityMap{
		"a": map[string]any{"loadId": "load-1", "weight": 11.0},
		"b": map[string]any{"loadId": "load-1", "weight": 22.0, "progress": "partial"},
		"c": map[string]any{"loadId": "load-1", "weight": 33.0, "progress": "done"},
		"d": map[string]any{"loadId": "load-1"},
	}
	got := lineLoadLeftoverIDs(domain)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got = %#v", got)
	}
}

func TestAdditionalPropertiesIsProgressAllowsNullAnyOf(t *testing.T) {
	if !additionalPropertiesIsProgress(map[string]any{"type": "string"}) {
		t.Fatal("plain string type")
	}
	if !additionalPropertiesIsProgress(map[string]any{
		"anyOf": []any{
			map[string]any{"type": "null"},
			map[string]any{"type": "string", "enum": []any{"partial", "done"}},
		},
	}) {
		t.Fatal("null|enum anyOf")
	}
	if additionalPropertiesIsProgress(map[string]any{"type": "object"}) {
		t.Fatal("object weigh cells must not classify as progress")
	}
}

func TestFilterFilledProgressKeepsPartialDoneAndNullClear(t *testing.T) {
	filled, err := filterFilledProgress(map[string]any{
		"a": "done", "b": "partial", "c": "", "d": nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filled) != 3 {
		t.Fatalf("filled = %#v", filled)
	}
	if filled["a"] != "done" || filled["b"] != "partial" || filled["d"] != nil {
		t.Fatalf("filled = %#v", filled)
	}
	if validateProgressFacts(filled, []string{"a", "b", "d"}) != "" {
		t.Fatal("valid batch must pass")
	}
	_, err = filterFilledProgress(map[string]any{"a": false})
	if err == nil || !strings.Contains(err.Error(), "invalid progress") {
		t.Fatalf("err = %v", err)
	}
}
