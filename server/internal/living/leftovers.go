package living

import (
	"fmt"
	"strings"
)

// filterFilledObjectRows keeps object cells that have every schema.required field
// filled, validating present properties against the row schema (enums, numbers).
func filterFilledObjectRows(value map[string]any, rowSchema map[string]any) (map[string]any, error) {
	props, _ := rowSchema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	required := schemaStringList(rowSchema["required"])
	filled := make(map[string]any)
	for id, cell := range value {
		if cell == nil {
			continue
		}
		row, ok := cell.(map[string]any)
		if !ok {
			if isEmptyScalar(cell) {
				continue
			}
			return nil, fmt.Errorf("invalid row for %s", id)
		}
		if objectRowUntouched(row, required) {
			continue
		}
		out, err := coerceObjectRow(row, props, required, id)
		if err != nil {
			return nil, err
		}
		filled[id] = out
	}
	return filled, nil
}

func objectRowUntouched(row map[string]any, required []string) bool {
	if len(required) == 0 {
		return len(row) == 0
	}
	for _, key := range required {
		raw, ok := row[key]
		if !ok || isEmptyScalar(raw) {
			return true
		}
	}
	return false
}

func coerceObjectRow(row map[string]any, props map[string]any, required []string, id string) (map[string]any, error) {
	out := make(map[string]any, len(row))
	req := make(map[string]struct{}, len(required))
	for _, key := range required {
		req[key] = struct{}{}
	}
	for key, raw := range row {
		prop, ok := props[key].(map[string]any)
		if !ok {
			out[key] = raw
			continue
		}
		if isEmptyScalar(raw) {
			if _, need := req[key]; need {
				return nil, fmt.Errorf("invalid %s for %s", key, id)
			}
			continue
		}
		coerced, err := coerceSchemaValue(raw, prop, key, id)
		if err != nil {
			return nil, err
		}
		out[key] = coerced
	}
	for _, key := range required {
		if _, ok := out[key]; !ok {
			return nil, fmt.Errorf("invalid %s for %s", key, id)
		}
	}
	return out, nil
}

func coerceSchemaValue(raw any, prop map[string]any, key, id string) (any, error) {
	if enums := schemaStringEnum(prop); len(enums) > 0 {
		s, ok := raw.(string)
		if !ok || !stringInList(s, enums) {
			return nil, fmt.Errorf("invalid %s for %s", key, id)
		}
		return s, nil
	}
	if schemaTypeIncludesNumber(prop["type"]) {
		n, ok := floatFromAny(raw)
		if !ok {
			return nil, fmt.Errorf("invalid %s for %s", key, id)
		}
		return n, nil
	}
	if schemaTypeIncludesString(prop["type"]) {
		s, ok := raw.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("invalid %s for %s", key, id)
		}
		return s, nil
	}
	return raw, nil
}

func validateObjectRowFacts(facts map[string]any, leftoverIDs []string, rowSchema map[string]any) string {
	if len(facts) == 0 {
		return "nothing to submit — fill at least one weight"
	}
	allowed := make(map[string]struct{}, len(leftoverIDs))
	for _, id := range leftoverIDs {
		allowed[id] = struct{}{}
	}
	props, _ := rowSchema["properties"].(map[string]any)
	for id, raw := range facts {
		if _, ok := allowed[id]; !ok {
			return fmt.Sprintf("pallet not in leftovers: %s", id)
		}
		row, _ := raw.(map[string]any)
		if row == nil {
			continue
		}
		for key, val := range row {
			prop, _ := props[key].(map[string]any)
			if prop == nil {
				continue
			}
			if n, ok := val.(float64); ok && numberViolatesBounds(n, prop) {
				return fmt.Sprintf("%s must be > 0 for %s", key, id)
			}
		}
	}
	return ""
}

func numberViolatesBounds(v float64, prop map[string]any) bool {
	if ex, ok := floatFromAny(prop["exclusiveMinimum"]); ok {
		return !(v > ex)
	}
	if min, ok := floatFromAny(prop["minimum"]); ok {
		return v < min
	}
	return false
}

func isEmptyScalar(cell any) bool {
	if cell == nil {
		return true
	}
	if s, ok := cell.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
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
// nil in the result means clear/unset entity progress (kept, not dropped).
func filterFilledProgress(value map[string]any, cellSchema map[string]any) (map[string]any, error) {
	enums := progressEnumsFromSchema(cellSchema)
	if len(enums) == 0 {
		enums = []string{"partial", "done"}
	}
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
		if !ok || !stringInList(s, enums) {
			return nil, fmt.Errorf("invalid progress for %s", id)
		}
		filled[id] = s
	}
	return filled, nil
}

func progressEnumsFromSchema(add map[string]any) []string {
	if add == nil {
		return nil
	}
	if enums := schemaStringEnum(add); len(enums) > 0 {
		return enums
	}
	anyOf, ok := add["anyOf"].([]any)
	if !ok {
		return nil
	}
	for _, branch := range anyOf {
		b, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		if enums := schemaStringEnum(b); len(enums) > 0 {
			return enums
		}
	}
	return nil
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

func schemaStringList(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, _ := item.(string)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func schemaStringEnum(schema map[string]any) []string {
	if schema == nil {
		return nil
	}
	return schemaStringList(schema["enum"])
}

func stringInList(s string, list []string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func schemaTypeIncludesNumber(raw any) bool {
	switch t := raw.(type) {
	case string:
		return t == "number" || t == "integer"
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && (s == "number" || s == "integer") {
				return true
			}
		}
	}
	return false
}

func additionalPropertiesSchema(task map[string]any, primary string) map[string]any {
	schema, _ := task["schema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	prop, _ := props[primary].(map[string]any)
	if prop == nil {
		return nil
	}
	add, _ := prop["additionalProperties"].(map[string]any)
	return add
}

// leftoverIDsForTask resolves leftover eligibility from Task reads.where (config-driven).
func leftoverIDsForTask(domain Domain, jobID string, task map[string]any, jobInstanceID string, catalogs map[string]any) ([]string, error) {
	read, _, ok, err := leftoverReadFromTask(task)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	label := jobInstanceLabel(domain, jobID, jobInstanceID)
	return ResolveRead(domain, read, jobInstanceID, label, catalogs)
}

// Test helpers kept as thin wrappers over fixture where exprs (no plant filters in Go).
func palletWeightLeftoverIDs(domain Domain, jobInstanceID string) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityPallet,
		Where:  "loadId = $instance.id and $not($exists(weight))",
		Key:    "id",
	}, jobInstanceID, "", nil)
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
	}, "", "", nil)
	if err != nil {
		return nil
	}
	return ids
}
