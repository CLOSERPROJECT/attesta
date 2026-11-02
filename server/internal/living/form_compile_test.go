package living

import "testing"

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
						"dualSources": map[string]any{
							"type":  "boolean",
							"title": "Fonti doppie",
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
	want := "Enter Peso…"
	if got != want {
		t.Fatalf("plastica.items.weight placeholder = %q, want %q", got, want)
	}
	if _, hasDual := items["dualSources"]; hasDual {
		t.Fatal("boolean dualSources must not receive a default placeholder")
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
			"sourceLoadId": map[string]any{
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

	source, _ := ui["sourceLoadId"].(map[string]any)
	if got := placeholderFromFieldUI(source); got != "Select Carico sorgente…" {
		t.Fatalf("enum placeholder = %q, want Select…", got)
	}
	note, _ := ui["note"].(map[string]any)
	if got := placeholderFromFieldUI(note); got != "Enter Nota…" {
		t.Fatalf("plain string placeholder = %q, want Enter…", got)
	}
}
