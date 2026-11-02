package living

import (
	"fmt"
	"strings"
)

type materialProp struct {
	Key   string
	Label string
}

type containerRegRow struct {
	Weight                float64
	Material              string
	SecondarySourceLoadID string // empty when dualSources is off
}

type containerRegFacts struct {
	SourceLoadID string
	Rows         []containerRegRow
}

// containerRegMaterials discovers weight-row array properties from schema.
// Requires sourceLoadId so unrelated array forms do not flatten. Labels = titles.
// Order follows authored uiSchema ui:options.order when present.
func containerRegMaterials(task map[string]any) ([]materialProp, bool) {
	schema, _ := task["schema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return nil, false
	}
	if _, ok := props["sourceLoadId"]; !ok {
		return nil, false
	}
	byKey := make(map[string]materialProp)
	for key, raw := range props {
		if key == "sourceLoadId" || key == "secondarySourceLoadId" {
			continue
		}
		prop, _ := raw.(map[string]any)
		if prop == nil {
			continue
		}
		if !schemaTypeIs(prop["type"], "array") {
			continue
		}
		items, _ := prop["items"].(map[string]any)
		if items == nil {
			continue
		}
		itemProps, _ := items["properties"].(map[string]any)
		if itemProps == nil {
			continue
		}
		if _, ok := itemProps["weight"]; !ok {
			continue
		}
		label, _ := prop["title"].(string)
		label = strings.TrimSpace(label)
		if label == "" {
			label = key
		}
		byKey[key] = materialProp{Key: key, Label: label}
	}
	if len(byKey) == 0 {
		return nil, false
	}
	out := make([]materialProp, 0, len(byKey))
	seen := make(map[string]struct{}, len(byKey))
	for _, key := range uiSchemaPropertyOrder(task) {
		mat, ok := byKey[key]
		if !ok {
			continue
		}
		out = append(out, mat)
		seen[key] = struct{}{}
	}
	for _, key := range sortedStringKeys(props) {
		if _, ok := seen[key]; ok {
			continue
		}
		if mat, ok := byKey[key]; ok {
			out = append(out, mat)
		}
	}
	return out, true
}

func uiSchemaPropertyOrder(task map[string]any) []string {
	ui, _ := task["uiSchema"].(map[string]any)
	if ui == nil {
		return nil
	}
	opts, _ := ui["ui:options"].(map[string]any)
	if opts == nil {
		return nil
	}
	return schemaStringList(opts["order"])
}

func schemaTypeIs(raw any, want string) bool {
	switch t := raw.(type) {
	case string:
		return t == want
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

// Flatten keyed material arrays → mintMany rows (experiment flattenContainerRegFacts).
func flattenContainerRegFacts(value map[string]any, materials []materialProp) (containerRegFacts, error) {
	sourceRaw, _ := value["sourceLoadId"].(string)
	sourceLoadID := strings.TrimSpace(sourceRaw)
	if sourceLoadID == "" {
		return containerRegFacts{}, fmt.Errorf("sourceLoadId is required")
	}

	secondaryRaw, _ := value["secondarySourceLoadId"].(string)
	formSecondary := strings.TrimSpace(secondaryRaw)

	rows := make([]containerRegRow, 0)
	for _, mat := range materials {
		arrRaw, exists := value[mat.Key]
		if !exists || arrRaw == nil {
			continue
		}
		arr, ok := arrRaw.([]any)
		if !ok {
			return containerRegFacts{}, fmt.Errorf("invalid %s — expected array", mat.Key)
		}
		for _, item := range arr {
			row, ok := item.(map[string]any)
			if !ok {
				return containerRegFacts{}, fmt.Errorf("invalid row in %s", mat.Label)
			}
			weightRaw, hasWeight := row["weight"]
			if !hasWeight || isEmptyScalar(weightRaw) {
				continue
			}
			weight, ok := floatFromAny(weightRaw)
			if !ok || !(weight > 0) {
				return containerRegFacts{}, fmt.Errorf("weight must be > 0 in %s", mat.Label)
			}

			dual, _ := row["dualSources"].(bool)
			if !dual {
				rows = append(rows, containerRegRow{Weight: weight, Material: mat.Label})
				continue
			}
			if formSecondary == "" {
				return containerRegFacts{}, fmt.Errorf("secondarySourceLoadId is required when any row has dualSources")
			}
			if formSecondary == sourceLoadID {
				return containerRegFacts{}, fmt.Errorf("secondarySourceLoadId must differ from sourceLoadId")
			}
			rows = append(rows, containerRegRow{
				Weight:                weight,
				Material:              mat.Label,
				SecondarySourceLoadID: formSecondary,
			})
		}
	}

	return containerRegFacts{SourceLoadID: sourceLoadID, Rows: rows}, nil
}

func containerRegFactsAsMap(facts containerRegFacts) map[string]any {
	rows := make([]any, 0, len(facts.Rows))
	for _, r := range facts.Rows {
		row := map[string]any{"weight": r.Weight, "material": r.Material}
		if r.SecondarySourceLoadID != "" {
			row["secondarySourceLoadId"] = r.SecondarySourceLoadID
		}
		rows = append(rows, row)
	}
	return map[string]any{
		"sourceLoadId": facts.SourceLoadID,
		"rows":         rows,
	}
}
