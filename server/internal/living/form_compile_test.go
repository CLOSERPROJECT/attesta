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
