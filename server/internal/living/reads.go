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
func ResolveRead(domain Domain, read ReadDef, jobInstanceID, jobInstanceLabel string, catalogs map[string]any) ([]string, error) {
	entities, err := listEntities(domain, read.Entity)
	if err != nil {
		return nil, err
	}
	bindings := domainBindings(domain, "", catalogs)
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

// leftoverReadFromTask extracts the Domain-read ReadDef from Task reads + dynamicProperties.
func leftoverReadFromTask(task map[string]any) (ReadDef, bool, error) {
	primary, ok := primaryDynamicProperty(task)
	if !ok {
		return ReadDef{}, false, nil
	}
	reads, _ := task["reads"].(map[string]any)
	if reads == nil {
		return ReadDef{}, false, fmt.Errorf("dynamic-map Task missing reads")
	}
	def, ok := dynamicPropertyDef(task, primary)
	if !ok || def.Read == "" {
		return ReadDef{}, false, fmt.Errorf("dynamic-map Task missing read key")
	}
	raw, ok := reads[def.Read]
	if !ok {
		return ReadDef{}, false, fmt.Errorf("reads.%s missing", def.Read)
	}
	read, err := parseReadDef(raw)
	if err != nil {
		return ReadDef{}, false, err
	}
	return read, true, nil
}

type dynamicPropDef struct {
	Read string
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
	if read == "" {
		return dynamicPropDef{}, false
	}
	return dynamicPropDef{Read: read}, true
}

func primaryDynamicProperty(task map[string]any) (key string, ok bool) {
	dyn, _ := task["dynamicProperties"].(map[string]any)
	if len(dyn) == 0 {
		return "", false
	}
	for _, k := range sortedStringKeys(dyn) {
		if _, ok := dynamicPropertyDef(task, k); ok {
			return k, true
		}
	}
	return "", false
}

func needsLeftoverProjection(task map[string]any) bool {
	_, ok := primaryDynamicProperty(task)
	return ok
}

// Full-map gate shared with prep: property in schema.required means every
// Domain-read id must be required so projected Ajv matches Commit.
func schemaRequiresProperty(task map[string]any, property string) bool {
	schema, _ := task["schema"].(map[string]any)
	if schema == nil {
		return false
	}
	for _, s := range schemaStringList(schema["required"]) {
		if s == property {
			return true
		}
	}
	// YAML/JSON may keep a typed string slice.
	if strs, ok := schema["required"].([]string); ok {
		for _, s := range strs {
			if s == property {
				return true
			}
		}
	}
	return false
}
