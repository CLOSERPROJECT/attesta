package living

import (
	"fmt"
	"strings"
)

// Fixed plant materials for containerReg — schema keys + human labels as entity facts.
var containerRegMaterials = []struct {
	Key   string
	Label string
}{
	{Key: "polvereDiVetro", Label: "Polvere di vetro"},
	{Key: "granellaDiVetro", Label: "Granella di vetro"},
	{Key: "granellaDiVetro2", Label: "Granella di vetro 2"},
	{Key: "polvereDiSilicio", Label: "Polvere di silicio"},
	{Key: "plastica", Label: "Plastica"},
	{Key: "alluminio", Label: "Alluminio"},
	{Key: "sopravaglio", Label: "Sopravaglio"},
	{Key: "polvereDiAspirazione", Label: "Polvere di aspirazione"},
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

// Flatten keyed material arrays → mintMany rows (experiment flattenContainerRegFacts).
func flattenContainerRegFacts(value map[string]any) (containerRegFacts, error) {
	sourceRaw, _ := value["sourceLoadId"].(string)
	sourceLoadID := strings.TrimSpace(sourceRaw)
	if sourceLoadID == "" {
		return containerRegFacts{}, fmt.Errorf("sourceLoadId is required")
	}

	secondaryRaw, _ := value["secondarySourceLoadId"].(string)
	formSecondary := strings.TrimSpace(secondaryRaw)

	rows := make([]containerRegRow, 0)
	for _, mat := range containerRegMaterials {
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
			if !hasWeight || isEmptyWeightCell(weightRaw) {
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
