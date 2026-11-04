package living

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TaskFormBundle is the minimum payload to mount one Task form without
// client FormPlan compile from full authored Task + full Domain.
type TaskFormBundle struct {
	JobID           string                 `json:"jobId"`
	TaskID          string                 `json:"taskId"`
	JobInstanceID   string                 `json:"jobInstanceId"`
	GroupID         string                 `json:"groupId,omitempty"`
	GroupInstanceID string                 `json:"groupInstanceId,omitempty"`
	Title           string                 `json:"title"`
	Surface         string                 `json:"surface"` // live
	Schema          map[string]any         `json:"schema"`
	UiSchema        map[string]any         `json:"uiSchema"`
	InitialData     map[string]any         `json:"initialData"`
	RowIDs          []string               `json:"rowIds,omitempty"`
	Columns         *FormColumns           `json:"columns,omitempty"`
	Chrome          []ChromeRule           `json:"chrome"`
	Derived         map[string]DerivedRule `json:"derived,omitempty"`
	ChromeFieldIDs  []string               `json:"chromeFieldIds"`
	SealFieldIDs    []string               `json:"sealFieldIds"`
	ProjectionKey   string                 `json:"projectionKey,omitempty"`
	Bindings        map[string]any         `json:"bindings"`
}

// FormColumns labels for row / map-table Id header.
type FormColumns struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// ChromeRule is a display chrome entry. dataReport is evaluated server-side into
// bindings.chromeDisplay; readonly stays client-eval for now.
type ChromeRule struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Expr   string `json:"expr,omitempty"`
	Title  string `json:"title,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// DerivedRule is a Derived key expr (concrete or expanded from map-star).
type DerivedRule struct {
	Expr        string `json:"expr"`
	Overridable *bool  `json:"overridable,omitempty"`
}

// BuildTaskFormBundle projects schema/ui + chrome/derived rules for one Task.
// Living code stays generic (no fixture Task id branches).
func BuildTaskFormBundle(cfg OrgConfig, domain Domain, ref ActionableTaskRef) (*TaskFormBundle, error) {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return nil, fmt.Errorf("living: unknown job %q", ref.JobID)
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil, fmt.Errorf("living: invalid job %q", ref.JobID)
	}
	pathRef := TaskPathRef{
		JobID:           ref.JobID,
		GroupID:         ref.GroupID,
		GroupInstanceID: ref.GroupInstanceID,
		TaskID:          ref.TaskID,
	}
	task, ok := LeafTaskByPath(job.Tasks, pathRef)
	if !ok {
		addr := ref.JobID + "/" + ref.TaskID
		if ref.GroupID != "" {
			addr = ref.JobID + "/" + ref.GroupID + "/" + ref.TaskID
		}
		return nil, fmt.Errorf("living: unknown task %s", addr)
	}
	payload := taskPayload(task)
	if payload == nil {
		return nil, fmt.Errorf("living: empty task payload %s/%s", ref.JobID, ref.TaskID)
	}

	projected, err := projectTaskForm(domain, payload, ref.JobInstanceID, jobInstanceLabel(domain, ref.JobID, ref.JobInstanceID), cfg.Catalogs)
	if err != nil {
		return nil, err
	}

	chrome := readChromeRules(payload)
	chromeRaw, _ := payload["chrome"].(map[string]any)
	sealIDs := authoredPropertyKeys(projected.Schema)

	for _, c := range chrome {
		if _, hit := sealIDs[c.ID]; hit {
			return nil, fmt.Errorf("living: chrome id %q collides with authored property", c.ID)
		}
	}

	schema := projected.Schema
	uiSchema := projected.UiSchema
	if len(chrome) > 0 {
		schema, uiSchema = mergeChromeIntoSchemas(schema, uiSchema, chrome, chromeRaw)
	}
	chromeFieldIDs := make([]string, len(chrome))
	for i, c := range chrome {
		chromeFieldIDs[i] = c.ID
	}
	uiSchema = withDefaultPlaceholders(uiSchema, schema, chromeFieldIDs)
	paintHideRootFormTitle(uiSchema)

	derived := compileDerivedRules(payload, projected.RowIDs)

	sealFieldIDs := sortedKeys(sealIDs)
	title := taskTitle(payload, ref.TaskID)
	projectionKey := ""
	if len(projected.RowIDs) > 0 {
		projectionKey = rowsIdentity(projected.RowIDs)
	}

	initial := seedInitialData(domain, payload, projected)
	bindings := buildFormBindings(domain, ref, cfg.Catalogs)
	if err := evaluateDataReports(chrome, bindings); err != nil {
		return nil, err
	}

	bundle := &TaskFormBundle{
		JobID:           ref.JobID,
		TaskID:          ref.TaskID,
		JobInstanceID:   ref.JobInstanceID,
		GroupID:         ref.GroupID,
		GroupInstanceID: ref.GroupInstanceID,
		Title:           title,
		Surface:         "live",
		Schema:          schema,
		UiSchema:        uiSchema,
		InitialData:     initial,
		RowIDs:          projected.RowIDs,
		Columns:         projected.Columns,
		Chrome:          chrome,
		Derived:         derived,
		ChromeFieldIDs:  chromeFieldIDs,
		SealFieldIDs:    sealFieldIDs,
		ProjectionKey:   projectionKey,
		Bindings:        bindings,
	}
	return bundle, nil
}

// evaluateDataReports runs each dataReport chrome expr into bindings.chromeDisplay[id].
// Input `$` is unused ({}); `$domain` / `$catalogs` (and `$instance` when present) come from bindings.
func evaluateDataReports(chrome []ChromeRule, bindings map[string]any) error {
	var reports []ChromeRule
	for _, c := range chrome {
		if c.Kind == "dataReport" {
			reports = append(reports, c)
		}
	}
	if len(reports) == 0 {
		return nil
	}
	domainBag, _ := bindings["domain"].(map[string]any)
	vars := ExprBindings{
		"domain":   domainBag,
		"catalogs": catalogsBag(nil),
	}
	if c, ok := bindings["catalogs"].(map[string]any); ok {
		vars["catalogs"] = c
	}
	if inst, ok := bindings["instance"]; ok {
		vars["instance"] = inst
	}
	display := map[string]any{}
	for _, c := range reports {
		raw, err := evaluateExpr(c.Expr, map[string]any{}, vars)
		if err != nil {
			return fmt.Errorf("living: dataReport %q: %w", c.ID, err)
		}
		dto, ok := raw.(map[string]any)
		if !ok || dto == nil {
			return fmt.Errorf("living: dataReport %q: expected object display DTO", c.ID)
		}
		if _, has := dto["title"]; !has && c.Title != "" {
			dto["title"] = c.Title
		}
		display[c.ID] = dto
	}
	bindings["chromeDisplay"] = display
	return nil
}

func taskTitle(payload map[string]any, taskID string) string {
	schema, _ := payload["schema"].(map[string]any)
	if schema != nil {
		if title, ok := schema["title"].(string); ok && title != "" {
			return title
		}
	}
	return taskID
}

func buildFormBindings(domain Domain, ref ActionableTaskRef, catalogs map[string]any) map[string]any {
	// DomainSnapshot-shaped bag (JSON field names) for Chrome / dataReport JSONata.
	subs := make([]any, len(domain.Submissions))
	for i, s := range domain.Submissions {
		subs[i] = deepCloneJSON(s)
	}
	maps := domain.EntityMaps()
	domainBag := make(map[string]any, len(maps)+1)
	for kind, m := range maps {
		domainBag[kind] = entityMapToAny(m)
	}
	domainBag["submissions"] = subs
	bindings := map[string]any{
		"domain":   domainBag,
		"catalogs": cloneMapJSON(catalogsBag(catalogs)),
	}
	if ref.JobInstanceID != "" {
		inst := map[string]any{"id": ref.JobInstanceID}
		if label := jobInstanceLabel(domain, ref.JobID, ref.JobInstanceID); label != "" {
			inst["label"] = label
		}
		bindings["instance"] = inst
	}
	return bindings
}

func seedInitialData(domain Domain, payload map[string]any, projected projectedForm) map[string]any {
	out := map[string]any{}
	primary, ok := primaryDynamicProperty(payload)
	if ok && len(projected.RowIDs) > 0 {
		add := additionalPropertiesSchema(payload, primary)
		if additionalPropertiesIsCellMap(add) {
			enums := schemaStringEnum(add)
			cells := map[string]any{}
			entities := rowEntityMap(domain, payload)
			for _, id := range projected.RowIDs {
				facts, hit := entities[id]
				if !hit {
					continue
				}
				p, _ := facts[primary].(string)
				if stringInList(p, enums) {
					cells[id] = p
				}
			}
			if len(cells) > 0 {
				out[primary] = cells
			}
		} else if schemaTypeIs(add["type"], "object") {
			if template := seedRowFromAuthoredDefaults(add); len(template) > 0 {
				rows := map[string]any{}
				for _, id := range projected.RowIDs {
					rows[id] = cloneMapJSON(template)
				}
				out[primary] = rows
			}
		}
	}
	seedFormDefaultsFromSchema(out, payload)
	return out
}

func rowEntityMap(domain Domain, payload map[string]any) EntityMap {
	read, ok, err := rowReadFromTask(payload)
	if err != nil || !ok {
		return EntityMap{}
	}
	m, err := listEntities(domain, read.Entity)
	if err != nil || m == nil {
		return EntityMap{}
	}
	return m
}

// Prefer omit over inventing first-enum values when a row has no authored defaults.
func seedRowFromAuthoredDefaults(rowSchema map[string]any) map[string]any {
	props, _ := rowSchema["properties"].(map[string]any)
	if props == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for key, raw := range props {
		prop, _ := raw.(map[string]any)
		if prop == nil {
			continue
		}
		if def, ok := prop["default"]; ok {
			out[key] = def
		}
	}
	return out
}

func seedFormDefaultsFromSchema(out map[string]any, payload map[string]any) {
	schema, _ := payload["schema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return
	}
	required := make(map[string]struct{})
	for _, key := range schemaStringList(schema["required"]) {
		required[key] = struct{}{}
	}
	for key, raw := range props {
		if _, exists := out[key]; exists {
			continue
		}
		prop, _ := raw.(map[string]any)
		if prop == nil || schemaTypeIs(prop["type"], "object") || schemaTypeIs(prop["type"], "array") {
			continue
		}
		if def, ok := prop["default"]; ok {
			out[key] = def
			continue
		}
		if _, need := required[key]; !need {
			continue
		}
		if enums := schemaStringEnum(prop); len(enums) > 0 {
			out[key] = enums[0]
		}
	}
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

func rowsIdentity(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func deepCloneJSON(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func cloneMapJSON(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	cloned := deepCloneJSON(m)
	out, _ := cloned.(map[string]any)
	if out == nil {
		return map[string]any{}
	}
	return out
}
