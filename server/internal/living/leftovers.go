package living

import (
	"fmt"
	"sort"
	"strings"
)

// WeighRowFact is one object-row leftover weigh cell after filter.
type WeighRowFact struct {
	Weight     float64
	TareKind   string
	Idoneita   string
	PanelCount float64
}

// palletWeightLeftoverIDs: pallets for this Job instance that still lack weight
// (fixture reads.leftovers where: loadId = $instance.id and $not($exists(weight))).
func palletWeightLeftoverIDs(domain Domain, jobInstanceID string) []string {
	ids := make([]string, 0)
	for id, facts := range domain.Pallet {
		loadID, _ := facts["loadId"].(string)
		if loadID != jobInstanceID {
			continue
		}
		if palletWeightExists(facts) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func palletWeightExists(facts map[string]any) bool {
	v, ok := facts["weight"]
	return ok && v != nil
}

func filterFilledObjectWeights(value map[string]any) (map[string]WeighRowFact, error) {
	filled := make(map[string]WeighRowFact)
	for id, cell := range value {
		if cell == nil {
			continue
		}
		row, ok := cell.(map[string]any)
		if !ok {
			if isEmptyWeightCell(cell) {
				continue
			}
			return nil, fmt.Errorf("invalid weight for %s", id)
		}
		weightRaw, hasWeight := row["weight"]
		if !hasWeight || isEmptyWeightCell(weightRaw) {
			continue
		}
		weight, ok := floatFromAny(weightRaw)
		if !ok {
			return nil, fmt.Errorf("invalid weight for %s", id)
		}
		panelCount, ok := floatFromAny(row["panelCount"])
		if !ok {
			return nil, fmt.Errorf("invalid panelCount for %s", id)
		}
		tareKind, ok := row["tareKind"].(string)
		if !ok || strings.TrimSpace(tareKind) == "" {
			return nil, fmt.Errorf("invalid tareKind for %s", id)
		}
		idoneita, ok := row["idoneita"].(string)
		if !ok || (idoneita != "NIR" && idoneita != "IR") {
			return nil, fmt.Errorf("invalid idoneita for %s", id)
		}
		filled[id] = WeighRowFact{
			Weight:     weight,
			TareKind:   tareKind,
			Idoneita:   idoneita,
			PanelCount: panelCount,
		}
	}
	return filled, nil
}

func isEmptyWeightCell(cell any) bool {
	if cell == nil {
		return true
	}
	if s, ok := cell.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

func validateWeighFacts(facts map[string]WeighRowFact, leftoverIDs []string) string {
	if len(facts) == 0 {
		return "nothing to submit — fill at least one weight"
	}
	allowed := make(map[string]struct{}, len(leftoverIDs))
	for _, id := range leftoverIDs {
		allowed[id] = struct{}{}
	}
	for id, row := range facts {
		if _, ok := allowed[id]; !ok {
			return fmt.Sprintf("pallet not in leftovers: %s", id)
		}
		if !(row.Weight > 0) {
			return fmt.Sprintf("weight must be > 0 for %s", id)
		}
	}
	return ""
}

// completeness all: every leftover id must appear in facts (empty leftover set → OK).
func enforceAllCompleteness(facts map[string]WeighRowFact, leftoverIDs []string) string {
	if len(leftoverIDs) == 0 {
		return ""
	}
	var missing []string
	for _, id := range leftoverIDs {
		if _, ok := facts[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "completeness all requires every leftover — missing " + strings.Join(missing, ", ")
}

func floatFromAny(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func weighFactsAsMap(facts map[string]WeighRowFact) map[string]any {
	out := make(map[string]any, len(facts))
	for id, row := range facts {
		out[id] = map[string]any{
			"weight":     row.Weight,
			"tareKind":   row.TareKind,
			"idoneita":   row.Idoneita,
			"panelCount": row.PanelCount,
		}
	}
	return out
}
