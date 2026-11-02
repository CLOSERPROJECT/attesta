package living

import (
	"fmt"
	"sort"
)

type projectedForm struct {
	Schema   map[string]any
	UiSchema map[string]any
	RowIDs   []string
	Columns  *FormColumns
}

func projectTaskForm(domain Domain, task map[string]any, jobInstanceID, jobInstanceLabel string, catalogs map[string]any) (projectedForm, error) {
	schemaRaw, _ := task["schema"].(map[string]any)
	if schemaRaw == nil {
		schemaRaw = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	schema := cloneMapJSON(schemaRaw)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
		schema["properties"] = props
	} else {
		props = cloneMapJSON(props)
		schema["properties"] = props
	}

	uiRaw, _ := task["uiSchema"].(map[string]any)
	uiSchema := cloneMapJSON(uiRaw)

	rowIDs := []string{}
	var columns *FormColumns
	primary, hasPrimary := primaryDynamicProperty(task)
	mutated := false

	dyn, _ := task["dynamicProperties"].(map[string]any)
	if len(dyn) > 0 {
		reads, _ := task["reads"].(map[string]any)
		if reads == nil {
			return projectedForm{}, fmt.Errorf("dynamicProperties requires reads")
		}
		keys := sortedStringKeys(dyn)
		for _, property := range keys {
			def, ok := dynamicPropertyDef(task, property)
			if !ok || def.Read == "" {
				return projectedForm{}, fmt.Errorf("dynamicProperties.%s is invalid", property)
			}
			raw, ok := reads[def.Read]
			if !ok {
				return projectedForm{}, fmt.Errorf("reads.%s missing", def.Read)
			}
			read, err := parseReadDef(raw)
			if err != nil {
				return projectedForm{}, err
			}
			ids, err := ResolveRead(domain, read, jobInstanceID, jobInstanceLabel, catalogs)
			if err != nil {
				return projectedForm{}, err
			}
			propSchema, _ := props[property].(map[string]any)
			if propSchema == nil {
				return projectedForm{}, fmt.Errorf("dynamicProperties.%s requires schema.properties.%s", property, property)
			}
			propSchema, err = materializeDynamicMapSchema(propSchema, ids, schemaRequiresProperty(task, property))
			if err != nil {
				return projectedForm{}, fmt.Errorf("dynamicProperties.%s: %w", property, err)
			}
			props[property] = propSchema
			mutated = true

			propUi, _ := uiSchema[property].(map[string]any)
			if propUi == nil {
				propUi = map[string]any{}
			}
			uiSchema[property] = stampConcreteMapUi(paintAuthoredFieldMarkers(propUi), ids)

			if hasPrimary && property == primary {
				rowIDs = ids
				columns = readFormColumns(uiSchema[property])
			}
		}
	}

	enumReads, _ := task["enumReads"].(map[string]any)
	if len(enumReads) > 0 {
		reads, _ := task["reads"].(map[string]any)
		if reads == nil {
			return projectedForm{}, fmt.Errorf("enumReads requires reads")
		}
		for _, property := range sortedStringKeys(enumReads) {
			readName, _ := enumReads[property].(string)
			if readName == "" {
				return projectedForm{}, fmt.Errorf("enumReads.%s must be a non-empty read name", property)
			}
			raw, ok := reads[readName]
			if !ok {
				return projectedForm{}, fmt.Errorf("reads.%s missing", readName)
			}
			read, err := parseReadDef(raw)
			if err != nil {
				return projectedForm{}, err
			}
			ids, err := ResolveRead(domain, read, jobInstanceID, jobInstanceLabel, catalogs)
			if err != nil {
				return projectedForm{}, err
			}
			if !injectEnumOnSchemaProperty(props, property, ids) {
				return projectedForm{}, fmt.Errorf("enumReads.%s requires schema.properties.%s or array items.properties.%s", property, property, property)
			}
			mutated = true
		}
	}

	_ = mutated
	rootOpts, _ := uiSchema["ui:options"].(map[string]any)
	if rootOpts == nil {
		rootOpts = map[string]any{}
	} else {
		rootOpts = cloneMapJSON(rootOpts)
	}
	rootOpts["expandable"] = false
	uiSchema["ui:options"] = rootOpts

	return projectedForm{
		Schema:   schema,
		UiSchema: uiSchema,
		RowIDs:   rowIDs,
		Columns:  columns,
	}, nil
}

// Authored additionalProperties is the per-id template; projection turns read
// ids into real properties so Ajv (and mapTable) see schema.properties rows.
func materializeDynamicMapSchema(propSchema map[string]any, ids []string, fullMap bool) (map[string]any, error) {
	next := cloneMapJSON(propSchema)
	template, ok := next["additionalProperties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("requires additionalProperties schema template")
	}
	concrete := make(map[string]any, len(ids))
	for _, id := range ids {
		concrete[id] = cloneMapJSON(template)
	}
	next["properties"] = concrete
	next["additionalProperties"] = false
	delete(next, "propertyNames")
	if fullMap {
		next["required"] = ids
	}
	return next, nil
}

func stampConcreteMapUi(propUi map[string]any, ids []string) map[string]any {
	next := cloneMapJSON(propUi)
	addUi, _ := next["additionalProperties"].(map[string]any)
	delete(next, "additionalProperties")
	if addUi == nil {
		return next
	}
	for _, id := range ids {
		next[id] = cloneMapJSON(addUi)
	}
	return next
}

func readFormColumns(propUi any) *FormColumns {
	cols := &FormColumns{ID: "Id", Value: "Value"}
	m, _ := propUi.(map[string]any)
	if m == nil {
		return cols
	}
	opts, _ := m["ui:options"].(map[string]any)
	if opts == nil {
		return cols
	}
	raw, _ := opts["columns"].(map[string]any)
	if raw == nil {
		return cols
	}
	if id, ok := raw["id"].(string); ok && id != "" {
		cols.ID = id
	}
	if value, ok := raw["value"].(string); ok && value != "" {
		cols.Value = value
	}
	return cols
}

func injectEnumOnSchemaProperty(schemaProperties map[string]any, property string, ids []string) bool {
	if prop, ok := schemaProperties[property].(map[string]any); ok {
		next := cloneMapJSON(prop)
		next["enum"] = ids
		schemaProperties[property] = next
		return true
	}
	found := false
	for key, raw := range schemaProperties {
		arr, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if arr["type"] != "array" {
			continue
		}
		items, _ := arr["items"].(map[string]any)
		if items == nil {
			continue
		}
		itemProps, _ := items["properties"].(map[string]any)
		if itemProps == nil {
			continue
		}
		nested, ok := itemProps[property].(map[string]any)
		if !ok {
			continue
		}
		nextArr := cloneMapJSON(arr)
		nextItems := cloneMapJSON(items)
		nextProps := cloneMapJSON(itemProps)
		nextNested := cloneMapJSON(nested)
		nextNested["enum"] = ids
		nextProps[property] = nextNested
		nextItems["properties"] = nextProps
		nextArr["items"] = nextItems
		schemaProperties[key] = nextArr
		found = true
	}
	return found
}

// Authored ui:field → sjsf ui:components (generic; not fixture Task ids).
var authoredFieldRegistry = map[string]struct {
	componentKey string
	slot         string
}{
	"mapTable":      {"mapTableField", "objectField"},
	"derivedLeaf":   {"derivedLeafField", "numberField"},
	"checkbox-only": {"checkboxOnlyField", "booleanField"},
}

func paintAuthoredFieldMarkers(uiNode map[string]any) map[string]any {
	next := cloneMapJSON(uiNode)
	if field, ok := next["ui:field"].(string); ok {
		if row, hit := authoredFieldRegistry[field]; hit {
			delete(next, "ui:field")
			existing, _ := next["ui:components"].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
			} else {
				existing = cloneMapJSON(existing)
			}
			existing[row.slot] = row.componentKey
			next["ui:components"] = existing
		}
	}
	if additional, ok := next["additionalProperties"].(map[string]any); ok {
		painted := map[string]any{}
		for k, child := range additional {
			if childMap, ok := child.(map[string]any); ok {
				painted[k] = paintAuthoredFieldMarkers(childMap)
			} else {
				painted[k] = child
			}
		}
		next["additionalProperties"] = painted
	}
	opts, _ := next["ui:options"].(map[string]any)
	if opts == nil {
		opts = map[string]any{}
	} else {
		opts = cloneMapJSON(opts)
	}
	opts["expandable"] = false
	next["ui:options"] = opts
	return next
}

func sortedStringKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
