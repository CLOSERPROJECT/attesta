package living

import (
	"fmt"
	"sort"
)

type ReadDef struct {
	Entity EntityType
	Where  string
	Key    string
}

func parseReadDef(raw any) (ReadDef, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return ReadDef{}, fmt.Errorf("read must be a mapping")
	}
	entity, _ := m["entity"].(string)
	if entity == "" {
		return ReadDef{}, fmt.Errorf("read.entity must be a non-empty string")
	}
	key, _ := m["key"].(string)
	if key == "" {
		return ReadDef{}, fmt.Errorf("read.key must be a non-empty string")
	}
	where, _ := m["where"].(string)
	return ReadDef{Entity: EntityType(entity), Where: where, Key: key}, nil
}

// ResolveRead returns entity ids matching an authored read (where + key).
func ResolveRead(domain Domain, read ReadDef, jobInstanceID, jobInstanceLabel string) ([]string, error) {
	entities, err := listEntities(domain, read.Entity)
	if err != nil {
		return nil, err
	}
	bindings := domainBindings(domain, "")
	if jobInstanceID != "" {
		inst := map[string]any{"id": jobInstanceID}
		if jobInstanceLabel != "" {
			inst["label"] = jobInstanceLabel
		}
		bindings["instance"] = inst
	}

	ids := make([]string, 0)
	for id, facts := range entities {
		if read.Where != "" {
			ok, err := evaluateExpr(read.Where, facts, bindings)
			if err != nil {
				return nil, err
			}
			if !truthy(ok) {
				continue
			}
		}
		if read.Key == "id" {
			ids = append(ids, id)
			continue
		}
		keyed, _ := facts[read.Key].(string)
		if keyed != "" {
			ids = append(ids, keyed)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func listEntities(domain Domain, entity EntityType) (EntityMap, error) {
	maps := domain.EntityMaps()
	m, ok := maps[string(entity)]
	if !ok || m == nil {
		return EntityMap{}, nil
	}
	return m, nil
}

func truthy(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return true
	}
}

func jobInstanceLabel(domain Domain, jobID, id string) string {
	for _, inst := range domain.JobInstances {
		if inst.JobID == jobID && inst.ID == id {
			return inst.Label
		}
	}
	return ""
}

// leftoverReadFromTask extracts the leftovers ReadDef from Task reads + dynamicProperties.
func leftoverReadFromTask(task map[string]any) (ReadDef, string, bool, error) {
	primary, completeness, ok := primaryDynamicProperty(task)
	if !ok {
		return ReadDef{}, "", false, nil
	}
	reads, _ := task["reads"].(map[string]any)
	if reads == nil {
		return ReadDef{}, "", false, fmt.Errorf("leftover-table Task missing reads")
	}
	def, ok := dynamicPropertyDef(task, primary)
	if !ok || def.Read == "" {
		return ReadDef{}, "", false, fmt.Errorf("leftover-table Task missing read key")
	}
	raw, ok := reads[def.Read]
	if !ok {
		return ReadDef{}, "", false, fmt.Errorf("reads.%s missing", def.Read)
	}
	read, err := parseReadDef(raw)
	if err != nil {
		return ReadDef{}, "", false, err
	}
	return read, completeness, true, nil
}

type dynamicPropDef struct {
	Read         string
	Completeness string
}

func dynamicPropertyDef(task map[string]any, key string) (dynamicPropDef, bool) {
	dyn, _ := task["dynamicProperties"].(map[string]any)
	if dyn == nil {
		return dynamicPropDef{}, false
	}
	raw, ok := dyn[key]
	if !ok {
		return dynamicPropDef{}, false
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return dynamicPropDef{}, false
	}
	read, _ := m["read"].(string)
	completeness, _ := m["completeness"].(string)
	return dynamicPropDef{Read: read, Completeness: completeness}, true
}

func primaryDynamicProperty(task map[string]any) (key string, completeness string, ok bool) {
	dyn, _ := task["dynamicProperties"].(map[string]any)
	if len(dyn) == 0 {
		return "", "", false
	}
	// Prefer authored order when present; map iteration is fine for truck-loop (one key).
	for k, raw := range dyn {
		m, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		comp, _ := m["completeness"].(string)
		if comp == "" {
			continue
		}
		return k, comp, true
	}
	return "", "", false
}

func needsLeftoverProjection(task map[string]any) bool {
	_, _, ok := primaryDynamicProperty(task)
	return ok
}
