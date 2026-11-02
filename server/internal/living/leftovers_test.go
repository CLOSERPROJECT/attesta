package living

import (
	"strings"
	"testing"
)

func feragameWeighRowSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"weight"},
		"properties": map[string]any{
			"weight":     map[string]any{"type": "number", "exclusiveMinimum": 0},
			"tareKind":   map[string]any{"type": "string", "enum": []any{"euro", "industrial", "custom"}},
			"idoneita":   map[string]any{"type": "string", "enum": []any{"NIR", "IR"}},
			"panelCount": map[string]any{"type": "number"},
		},
	}
}

func TestPalletWeightLeftoverIDsSkipsWeighted(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, kindPallet, EntityMap{
		"load-1.1": map[string]any{"loadId": "load-1"},
		"load-1.2": map[string]any{"loadId": "load-1", "weight": 11.0},
		"load-2.1": map[string]any{"loadId": "load-2"},
	})
	got := palletWeightLeftoverIDs(domain, "load-1")
	if len(got) != 1 || got[0] != "load-1.1" {
		t.Fatalf("got = %#v", got)
	}
}

func TestEnforceRequiredMapKeysPartial(t *testing.T) {
	facts := map[string]any{"a": map[string]any{"weight": 1.0}}
	msg := enforceRequiredMapKeys(facts, []string{"a", "b"})
	if msg == "" || !strings.Contains(msg, "b") {
		t.Fatalf("msg = %q", msg)
	}
	if enforceRequiredMapKeys(facts, []string{"a"}) != "" {
		t.Fatal("complete map must pass")
	}
}

func TestFilterFilledObjectRowsUsesSchema(t *testing.T) {
	schema := feragameWeighRowSchema()
	filled, err := filterFilledObjectRows(map[string]any{
		"p1": map[string]any{"weight": 11.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0},
		"p2": map[string]any{},
		"p3": map[string]any{"tareKind": "euro"}, // required weight missing → untouched
	}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(filled) != 1 {
		t.Fatalf("filled = %#v", filled)
	}

	_, err = filterFilledObjectRows(map[string]any{
		"p1": map[string]any{"weight": 11.0, "idoneita": "ZZ"},
	}, schema)
	if err == nil || !strings.Contains(err.Error(), "idoneita") {
		t.Fatalf("err = %v", err)
	}
}

func TestLineLoadLeftoverIDsSkipsDone(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, kindPallet, EntityMap{
		"a": map[string]any{"loadId": "load-1", "weight": 11.0},
		"b": map[string]any{"loadId": "load-1", "weight": 22.0, "progress": "partial"},
		"c": map[string]any{"loadId": "load-1", "weight": 33.0, "progress": "done"},
		"d": map[string]any{"loadId": "load-1"},
	})
	got := lineLoadLeftoverIDs(domain)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got = %#v", got)
	}
}

func TestLeftoverCoercionSchemaAndHelperEdges(t *testing.T) {
	_, err := coerceObjectRow(
		map[string]any{"weight": "  ", "note": "x"},
		map[string]any{"weight": map[string]any{"type": "number"}},
		[]string{"weight"},
		"p1",
	)
	if err == nil || !strings.Contains(err.Error(), "weight") {
		t.Fatalf("empty required scalar = %v", err)
	}
	_, err = coerceObjectRow(
		map[string]any{"extra": "keep"},
		map[string]any{"weight": map[string]any{"type": "number"}},
		[]string{"weight"},
		"p1",
	)
	if err == nil || !strings.Contains(err.Error(), "weight") {
		t.Fatalf("missing required = %v", err)
	}

	if _, err := coerceSchemaValue(1, map[string]any{"type": "string"}, "label", "p1"); err == nil {
		t.Fatal("non-string for string type")
	}
	if _, err := coerceSchemaValue("  ", map[string]any{"type": "string"}, "label", "p1"); err == nil {
		t.Fatal("blank string")
	}
	got, err := coerceSchemaValue(true, map[string]any{"type": "boolean"}, "flag", "p1")
	if err != nil || got != true {
		t.Fatalf("passthrough = %#v %v", got, err)
	}
	if !schemaTypeIncludesNumber([]any{"string", "integer"}) || schemaTypeIncludesNumber([]any{"string"}) || schemaTypeIncludesNumber(1) {
		t.Fatal("schemaTypeIncludesNumber branches")
	}
	if schemaStringEnum(nil) != nil {
		t.Fatal("nil schemaStringEnum")
	}
	if progressEnumsFromSchema(nil) != nil {
		t.Fatal("nil progressEnums")
	}
	if progressEnumsFromSchema(map[string]any{"anyOf": []any{"x", map[string]any{"type": "null"}}}) != nil {
		t.Fatal("anyOf without enums")
	}
	if numberViolatesBounds(1, map[string]any{"minimum": 2}) != true || numberViolatesBounds(3, map[string]any{"minimum": 2}) {
		t.Fatal("minimum bounds")
	}
	if !isEmptyScalar(nil) {
		t.Fatal("nil empty scalar")
	}
	if msg := validateObjectRowFacts(map[string]any{"ok": "skip"}, []string{"ok"}, map[string]any{"properties": map[string]any{}}); msg != "" {
		t.Fatalf("non-map row = %q", msg)
	}
	if additionalPropertiesSchema(map[string]any{
		"schema": map[string]any{"properties": map[string]any{}},
	}, "weights") != nil {
		t.Fatal("missing prop schema")
	}
	ids, err := leftoverIDsForTask(EmptyDomain(), "intake", map[string]any{}, "", nil)
	if err != nil || ids != nil {
		t.Fatalf("no dynamic leftovers = %#v %v", ids, err)
	}
	if !schemaRequiresProperty(map[string]any{
		"schema": map[string]any{"required": []string{"weights"}},
	}, "weights") || schemaRequiresProperty(map[string]any{}, "weights") {
		t.Fatal("schemaRequiresProperty typed required")
	}
	if !schemaTypeIs([]any{"object", "null"}, "object") || schemaTypeIs(1, "object") {
		t.Fatal("schemaTypeIs array/default")
	}
	if len(leftoverEntityMap(EmptyDomain(), map[string]any{})) != 0 {
		t.Fatal("leftoverEntityMap without read")
	}
	if len(seedRowFromSchema(map[string]any{})) != 0 {
		t.Fatal("seedRow empty props")
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
	schema := map[string]any{
		"anyOf": []any{
			map[string]any{"type": "null"},
			map[string]any{"type": "string", "enum": []any{"partial", "done"}},
		},
	}
	filled, err := filterFilledProgress(map[string]any{
		"a": "done", "b": "partial", "c": "", "d": nil,
	}, schema)
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
	_, err = filterFilledProgress(map[string]any{"a": false}, schema)
	if err == nil || !strings.Contains(err.Error(), "invalid progress") {
		t.Fatalf("err = %v", err)
	}
}
