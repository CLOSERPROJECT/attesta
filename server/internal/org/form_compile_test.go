package org

import (
	"encoding/json"
	"testing"
)

func TestMergeChromeOmitsEmptyRequired(t *testing.T) {
	chromeRaw := map[string]any{
		"hint": map[string]any{
			"kind":  "readonly",
			"expr":  "1",
			"title": "Hint",
		},
	}
	rules := readChromeRules(map[string]any{"chrome": chromeRaw})
	schema, _ := mergeChromeIntoSchemas(
		map[string]any{"type": "object", "properties": map[string]any{}},
		map[string]any{},
		rules,
		chromeRaw,
	)
	if _, has := schema["required"]; has {
		t.Fatalf("required must be omitted when empty; got %#v", schema["required"])
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, has := wire["required"]; has {
		t.Fatalf("JSON required must be absent (not null); got %#v in %s", wire["required"], raw)
	}
}

func TestMergeChromePreservesAuthoredRequired(t *testing.T) {
	chromeRaw := map[string]any{
		"hint": map[string]any{"kind": "readonly", "expr": "1", "title": "Hint"},
	}
	rules := readChromeRules(map[string]any{"chrome": chromeRaw})
	schema, _ := mergeChromeIntoSchemas(
		map[string]any{
			"type":       "object",
			"required":   []any{"name"},
			"properties": map[string]any{"name": map[string]any{"type": "string"}},
		},
		map[string]any{},
		rules,
		chromeRaw,
	)
	req, _ := schema["required"].([]any)
	if len(req) != 1 || req[0] != "name" {
		t.Fatalf("required = %#v, want [name]", req)
	}
}

func TestMergeChromeHideIfNullStamp(t *testing.T) {
	chromeRaw := map[string]any{
		"shown": map[string]any{
			"kind": "readonly",
			"expr": "1",
			"title": "Shown",
		},
		"hidden": map[string]any{
			"kind":       "readonly",
			"expr":       "1",
			"title":      "Hidden",
			"hideIfNull": true,
		},
		"explicitFalse": map[string]any{
			"kind":       "readonly",
			"expr":       "1",
			"title":      "False",
			"hideIfNull": false,
		},
	}
	rules := readChromeRules(map[string]any{"chrome": chromeRaw})
	byID := make(map[string]ChromeRule, len(rules))
	for _, r := range rules {
		byID[r.ID] = r
	}
	if !byID["hidden"].HideIfNull {
		t.Fatal("hidden rule must read hideIfNull: true")
	}
	if byID["shown"].HideIfNull || byID["explicitFalse"].HideIfNull {
		t.Fatalf("shown/explicitFalse must be false; got shown=%v explicitFalse=%v",
			byID["shown"].HideIfNull, byID["explicitFalse"].HideIfNull)
	}

	_, ui := mergeChromeIntoSchemas(map[string]any{}, map[string]any{}, rules, chromeRaw)
	for _, id := range []string{"shown", "explicitFalse"} {
		field, _ := ui[id].(map[string]any)
		opts, _ := field["ui:options"].(map[string]any)
		if _, ok := opts["hideIfNull"]; ok {
			t.Fatalf("%s ui:options must omit hideIfNull; got %#v", id, opts)
		}
		if opts["chromeId"] != id {
			t.Fatalf("%s chromeId = %#v", id, opts["chromeId"])
		}
	}
	hidden, _ := ui["hidden"].(map[string]any)
	opts, _ := hidden["ui:options"].(map[string]any)
	if opts["hideIfNull"] != true {
		t.Fatalf("hidden ui:options.hideIfNull = %#v, want true", opts["hideIfNull"])
	}
	if opts["chromeId"] != "hidden" {
		t.Fatalf("hidden chromeId = %#v", opts["chromeId"])
	}
}

func TestMergeChromePlaceholderStamp(t *testing.T) {
	chromeRaw := map[string]any{
		"net": map[string]any{
			"kind":        "readonly",
			"expr":        "1",
			"title":       "Net",
			"placeholder": "Calcolato…",
		},
	}
	rules := readChromeRules(map[string]any{"chrome": chromeRaw})
	_, ui := mergeChromeIntoSchemas(map[string]any{}, map[string]any{}, rules, chromeRaw)
	field, _ := ui["net"].(map[string]any)
	opts, _ := field["ui:options"].(map[string]any)
	if opts["placeholder"] != "Calcolato…" {
		t.Fatalf("placeholder = %#v, want Calcolato…", opts["placeholder"])
	}
}

func placeholderFromFieldUI(fieldUi map[string]any) string {
	if fieldUi == nil {
		return ""
	}
	if s, ok := fieldUi["ui:placeholder"].(string); ok {
		return s
	}
	opts, _ := fieldUi["ui:options"].(map[string]any)
	if opts == nil {
		return ""
	}
	if s, ok := opts["placeholder"].(string); ok {
		return s
	}
	for _, key := range []string{"shadcn4Text", "shadcn4Number", "shadcn4Select"} {
		widget, _ := opts[key].(map[string]any)
		if widget == nil {
			continue
		}
		if s, ok := widget["placeholder"].(string); ok {
			return s
		}
	}
	return ""
}

func TestWithDefaultPlaceholdersNestedItemsWeight(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plastica": map[string]any{
				"type":  "array",
				"title": "Plastica",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"weight": map[string]any{
							"type":  "number",
							"title": "Peso",
						},
						"incomplete": map[string]any{
							"type":  "boolean",
							"title": "Incompleto",
						},
					},
				},
			},
		},
	}

	ui := withDefaultPlaceholders(map[string]any{}, schema, nil)

	plastica, _ := ui["plastica"].(map[string]any)
	items, _ := plastica["items"].(map[string]any)
	weight, _ := items["weight"].(map[string]any)
	got := placeholderFromFieldUI(weight)
	want := "Inserisci Peso…"
	if got != want {
		t.Fatalf("plastica.items.weight placeholder = %q, want %q", got, want)
	}
	if _, hasBool := items["incomplete"]; hasBool {
		t.Fatal("boolean incomplete must not receive a default placeholder")
	}
}

func TestWithDefaultPlaceholdersAuthoredWins(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plastica": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"weight": map[string]any{
							"type":  "number",
							"title": "Peso",
						},
					},
				},
			},
			"name": map[string]any{
				"type":  "string",
				"title": "Nome",
			},
		},
	}
	authored := map[string]any{
		"name": map[string]any{
			"ui:placeholder": "Nome custom…",
		},
		"plastica": map[string]any{
			"items": map[string]any{
				"weight": map[string]any{
					"ui:options": map[string]any{
						"shadcn4Number": map[string]any{
							"placeholder": "Peso Plastica (kg)…",
						},
					},
				},
			},
		},
	}

	ui := withDefaultPlaceholders(authored, schema, nil)

	name, _ := ui["name"].(map[string]any)
	if got := placeholderFromFieldUI(name); got != "Nome custom…" {
		t.Fatalf("root name placeholder = %q, want authored", got)
	}
	plastica, _ := ui["plastica"].(map[string]any)
	items, _ := plastica["items"].(map[string]any)
	weight, _ := items["weight"].(map[string]any)
	if got := placeholderFromFieldUI(weight); got != "Peso Plastica (kg)…" {
		t.Fatalf("nested weight placeholder = %q, want authored", got)
	}
}

func TestWithDefaultPlaceholdersEnumSelect(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sourceLotId": map[string]any{
				"type":  "string",
				"title": "Carico sorgente",
				"enum":  []any{"L1", "L2"},
			},
			"note": map[string]any{
				"type":  "string",
				"title": "Nota",
			},
		},
	}

	ui := withDefaultPlaceholders(map[string]any{}, schema, nil)

	source, _ := ui["sourceLotId"].(map[string]any)
	if got := placeholderFromFieldUI(source); got != "Seleziona Carico sorgente…" {
		t.Fatalf("enum placeholder = %q, want Seleziona…", got)
	}
	note, _ := ui["note"].(map[string]any)
	if got := placeholderFromFieldUI(note); got != "Inserisci Nota…" {
		t.Fatalf("plain string placeholder = %q, want Inserisci…", got)
	}
}

func TestWithDefaultPlaceholdersExpandsAuthoredToShadcn(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"progress": map[string]any{
				"enum": []any{"partial", "done"},
			},
		},
	}
	authored := map[string]any{
		"progress": map[string]any{
			"ui:options": map[string]any{
				"placeholder": "–",
			},
		},
	}

	ui := withDefaultPlaceholders(authored, schema, nil)
	progress, _ := ui["progress"].(map[string]any)
	opts, _ := progress["ui:options"].(map[string]any)
	if opts["placeholder"] != "–" {
		t.Fatalf("opts.placeholder = %#v, want –", opts["placeholder"])
	}
	shadcn, _ := opts["shadcn4Select"].(map[string]any)
	if shadcn["placeholder"] != "–" {
		t.Fatalf("shadcn4Select.placeholder = %#v, want – (expanded from authored)", shadcn["placeholder"])
	}
}

func TestExpandFormLayoutColsAndSpan(t *testing.T) {
	ui := map[string]any{
		"ui:options": map[string]any{
			"layout": "cols-2",
			"order":  []any{"a", "notes"},
		},
		"notes": map[string]any{
			"ui:options": map[string]any{
				"layoutSpan": true,
			},
		},
		"next": map[string]any{
			"ui:options": map[string]any{
				"layoutNewRow": true,
			},
		},
	}
	expandFormLayout(ui)

	rootOpts, _ := ui["ui:options"].(map[string]any)
	if _, ok := rootOpts["layout"]; ok {
		t.Fatalf("root layout shortcut must be stripped; got %#v", rootOpts["layout"])
	}
	layouts, _ := rootOpts["layouts"].(map[string]any)
	props, _ := layouts["object-properties"].(map[string]any)
	if props["class"] != "form-cols-2" {
		t.Fatalf("object-properties class = %#v, want form-cols-2", props["class"])
	}

	notes, _ := ui["notes"].(map[string]any)
	notesOpts, _ := notes["ui:options"].(map[string]any)
	if _, ok := notesOpts["layoutSpan"]; ok {
		t.Fatalf("layoutSpan must be stripped; got %#v", notesOpts["layoutSpan"])
	}
	noteLayouts, _ := notesOpts["layouts"].(map[string]any)
	prop, _ := noteLayouts["object-property"].(map[string]any)
	if prop["class"] != "form-cols-span" {
		t.Fatalf("object-property class = %#v, want form-cols-span", prop["class"])
	}

	next, _ := ui["next"].(map[string]any)
	nextOpts, _ := next["ui:options"].(map[string]any)
	if _, ok := nextOpts["layoutNewRow"]; ok {
		t.Fatalf("layoutNewRow must be stripped; got %#v", nextOpts["layoutNewRow"])
	}
	nextLayouts, _ := nextOpts["layouts"].(map[string]any)
	nextProp, _ := nextLayouts["object-property"].(map[string]any)
	if nextProp["class"] != "form-cols-new-row" {
		t.Fatalf("object-property class = %#v, want form-cols-new-row", nextProp["class"])
	}
}

func TestExpandFormLayoutIdempotentClassAndLeavesAttrLayout(t *testing.T) {
	ui := map[string]any{
		"ui:options": map[string]any{
			"layout": "cols-3",
			"layouts": map[string]any{
				"object-properties": map[string]any{
					"class": "form-cols-3 extra",
				},
			},
		},
		"other": map[string]any{
			"ui:options": map[string]any{
				// sjsf attribute override shape — must not be touched
				"layout": map[string]any{"data-x": "1"},
			},
		},
	}
	expandFormLayout(ui)
	rootOpts, _ := ui["ui:options"].(map[string]any)
	layouts, _ := rootOpts["layouts"].(map[string]any)
	props, _ := layouts["object-properties"].(map[string]any)
	if props["class"] != "form-cols-3 extra" {
		t.Fatalf("class merge = %#v, want form-cols-3 extra", props["class"])
	}
	other, _ := ui["other"].(map[string]any)
	otherOpts, _ := other["ui:options"].(map[string]any)
	if _, ok := otherOpts["layout"].(map[string]any); !ok {
		t.Fatalf("object layout attrs must remain; got %#v", otherOpts["layout"])
	}
}
