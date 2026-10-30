package living

import (
	"fmt"
	"strings"
)

// WeighRowFact is one object-row leftover weigh cell after filter.
type WeighRowFact struct {
	Weight     float64
	TareKind   string
	Idoneita   string
	PanelCount float64
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
	asMap := make(map[string]any, len(facts))
	for id := range facts {
		asMap[id] = struct{}{}
	}
	return enforceAllCompletenessKeys(asMap, leftoverIDs)
}

// Untouched this pass — omit from batch (not the same as clear null).
func isUntouchedProgressCell(cell any) bool {
	if cell == nil {
		return false
	}
	if s, ok := cell.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

// Set or clear progress cells; untouched rows stay out of the batch.
// nil in the result means clear/unset entity progress.
func filterFilledProgress(value map[string]any) (map[string]any, error) {
	filled := make(map[string]any)
	for id, cell := range value {
		if isUntouchedProgressCell(cell) {
			continue
		}
		if cell == nil {
			filled[id] = nil
			continue
		}
		s, ok := cell.(string)
		if !ok || (s != "partial" && s != "done") {
			return nil, fmt.Errorf("invalid progress for %s", id)
		}
		filled[id] = s
	}
	return filled, nil
}

func validateProgressFacts(facts map[string]any, leftoverIDs []string) string {
	if len(facts) == 0 {
		return "nothing to submit — set or clear progress on at least one pallet"
	}
	allowed := make(map[string]struct{}, len(leftoverIDs))
	for _, id := range leftoverIDs {
		allowed[id] = struct{}{}
	}
	for id := range facts {
		if _, ok := allowed[id]; !ok {
			return fmt.Sprintf("pallet not in leftovers: %s", id)
		}
	}
	return ""
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

// leftoverIDsForTask resolves leftover eligibility from Task reads.where (config-driven).
func leftoverIDsForTask(domain Domain, jobID string, task map[string]any, jobInstanceID string) ([]string, error) {
	read, _, ok, err := leftoverReadFromTask(task)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	label := jobInstanceLabel(domain, jobID, jobInstanceID)
	return ResolveRead(domain, read, jobInstanceID, label)
}

// Test helpers kept as thin wrappers over fixture where exprs (no plant filters in Go).
func palletWeightLeftoverIDs(domain Domain, jobInstanceID string) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityPallet,
		Where:  "loadId = $instance.id and $not($exists(weight))",
		Key:    "id",
	}, jobInstanceID, "")
	if err != nil {
		return nil
	}
	return ids
}

func lineLoadLeftoverIDs(domain Domain) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityPallet,
		Where:  "$exists(weight) and $not(progress = 'done')",
		Key:    "id",
	}, "", "")
	if err != nil {
		return nil
	}
	return ids
}
