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

func TestPalletWeightRowIDsSkipsWeighted(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "pallet", EntityMap{
		"load-1.1": map[string]any{"loadId": "load-1"},
		"load-1.2": map[string]any{"loadId": "load-1", "weight": 11.0},
		"load-2.1": map[string]any{"loadId": "load-2"},
	})
	got := palletWeightRowIDs(domain, "load-1")
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

func TestLineLoadRowIDsSkipsDone(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "pallet", EntityMap{
		"a": map[string]any{"loadId": "load-1", "weight": 11.0},
		"b": map[string]any{"loadId": "load-1", "weight": 22.0, "progress": "partial"},
		"c": map[string]any{"loadId": "load-1", "weight": 33.0, "progress": "done"},
		"d": map[string]any{"loadId": "load-1"},
	})
	got := lineLoadRowIDs(domain)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got = %#v", got)
	}
}

func TestRowCoercionSchemaAndHelperEdges(t *testing.T) {
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
	if schemaStringEnum(map[string]any{
		"anyOf": []any{map[string]any{"type": "string", "enum": []any{"partial", "done"}}},
	}) != nil {
		t.Fatal("anyOf enums ignored")
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
	ids, err := rowIDsForTask(EmptyDomain(), "intake", map[string]any{}, "", nil)
	if err != nil || ids != nil {
		t.Fatalf("no dynamic rows = %#v %v", ids, err)
	}
	if !schemaRequiresProperty(map[string]any{
		"schema": map[string]any{"required": []string{"weights"}},
	}, "weights") || schemaRequiresProperty(map[string]any{}, "weights") {
		t.Fatal("schemaRequiresProperty typed required")
	}
	if !schemaTypeIs([]any{"object", "null"}, "object") || schemaTypeIs(1, "object") {
		t.Fatal("schemaTypeIs array/default")
	}
	if len(rowEntityMap(EmptyDomain(), map[string]any{})) != 0 {
		t.Fatal("rowEntityMap without read")
	}
	if len(seedRowFromAuthoredDefaults(map[string]any{})) != 0 {
		t.Fatal("seedRow empty props")
	}
	if got := seedRowFromAuthoredDefaults(map[string]any{
		"properties": map[string]any{
			"tareKind": map[string]any{"type": "string", "enum": []any{"euro", "industrial"}, "default": "euro"},
			"idoneita": map[string]any{"type": "string", "enum": []any{"NIR", "IR"}},
		},
	}); len(got) != 1 || got["tareKind"] != "euro" {
		t.Fatalf("authored default only = %#v", got)
	}
}

func TestAdditionalPropertiesIsCellMap(t *testing.T) {
	if !additionalPropertiesIsCellMap(map[string]any{"type": "string"}) {
		t.Fatal("plain string type")
	}
	if !additionalPropertiesIsCellMap(map[string]any{
		"enum": []any{"partial", "done"},
	}) {
		t.Fatal("string enum")
	}
	if additionalPropertiesIsCellMap(map[string]any{
		"anyOf": []any{
			map[string]any{"type": "null"},
			map[string]any{"type": "string", "enum": []any{"partial", "done"}},
		},
	}) {
		t.Fatal("anyOf must not classify as cell map")
	}
	if additionalPropertiesIsCellMap(map[string]any{"type": "object"}) {
		t.Fatal("object weigh cells must not classify as cell map")
	}
}

func TestFilterFilledCellsKeepsPartialDoneOmitsNilAndBlank(t *testing.T) {
	schema := map[string]any{
		"enum": []any{"partial", "done"},
	}
	filled, err := filterFilledCells(map[string]any{
		"a": "done", "b": "partial", "c": "", "d": nil,
	}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(filled) != 2 {
		t.Fatalf("filled = %#v", filled)
	}
	if filled["a"] != "done" || filled["b"] != "partial" {
		t.Fatalf("filled = %#v", filled)
	}
	if _, has := filled["d"]; has {
		t.Fatalf("nil must omit, not Domain clear: %#v", filled)
	}
	if validateCellFacts(filled, []string{"a", "b"}) != "" {
		t.Fatal("valid batch must pass")
	}
	_, err = filterFilledCells(map[string]any{"a": false}, schema)
	if err == nil || !strings.Contains(err.Error(), "invalid cell") {
		t.Fatalf("err = %v", err)
	}
}

// Fixture where wrappers (tests only — no fixture filters in product Go).
func palletWeightRowIDs(domain Domain, jobInstanceID string) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityType("pallet"),
		Where:  "loadId = $instance.id and $not($exists(weight))",
		Key:    "id",
	}, jobInstanceID, "", nil)
	if err != nil {
		return nil
	}
	return ids
}

func lineLoadRowIDs(domain Domain) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityType("pallet"),
		Where:  "$exists(weight) and $not(progress = 'done')",
		Key:    "id",
	}, "", "", nil)
	if err != nil {
		return nil
	}
	return ids
}
