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
	JobID          string                 `json:"jobId"`
	TaskID         string                 `json:"taskId"`
	JobInstanceID  string                 `json:"jobInstanceId"`
	Title          string                 `json:"title"`
	Surface        string                 `json:"surface"` // live | leftover-table
	Schema         map[string]any         `json:"schema"`
	UiSchema       map[string]any         `json:"uiSchema"`
	InitialData    map[string]any         `json:"initialData"`
	LeftoverIDs    []string               `json:"leftoverIds,omitempty"`
	Columns        *FormColumns           `json:"columns,omitempty"`
	Chrome         []ChromeRule           `json:"chrome"`
	Derived        map[string]DerivedRule `json:"derived,omitempty"`
	ChromeFieldIDs []string               `json:"chromeFieldIds"`
	SealFieldIDs   []string               `json:"sealFieldIds"`
	ProjectionKey  string                 `json:"projectionKey,omitempty"`
	Bindings       map[string]any         `json:"bindings"`
}

// FormColumns labels for leftover / map-table Id header.
type FormColumns struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// ChromeRule is a display chrome entry (rules only; live eval stays client-side).
type ChromeRule struct {
	ID     string          `json:"id"`
	Kind   string          `json:"kind"`
	Expr   string          `json:"expr,omitempty"`
	Title  string          `json:"title,omitempty"`
	Before string          `json:"before,omitempty"`
	After  string          `json:"after,omitempty"`
	Rows   []ChromeRecapRow `json:"rows,omitempty"`
}

// ChromeRecapRow is one recap row rule.
type ChromeRecapRow struct {
	Label string           `json:"label"`
	Expr  string           `json:"expr"`
	Tone  string           `json:"tone,omitempty"`
	Bands []ChromeToneBand `json:"bands,omitempty"`
}

// ChromeToneBand is a numeric tone threshold.
type ChromeToneBand struct {
	Min  float64 `json:"min"`
	Tone string  `json:"tone"`
}

// DerivedRule is a Derived key expr (concrete or expanded from map-star).
type DerivedRule struct {
	Expr        string `json:"expr"`
	Overridable *bool  `json:"overridable,omitempty"`
}

// AvailableTaskRef addresses one Worklist-available Task on an instance.
type AvailableTaskRef struct {
	JobID         string
	TaskID        string
	JobInstanceID string
}

// CollectAvailableTaskRefs walks Worklist in Org Config Job order: provisional
// first Task when canStart (empty jobInstanceId), then every available Task on
// each instance in authored Task order.
func CollectAvailableTaskRefs(cfg OrgConfig, wl Worklist) []AvailableTaskRef {
	byJob := make(map[string]WorklistJob, len(wl.Jobs))
	for _, j := range wl.Jobs {
		byJob[j.JobID] = j
	}
	out := make([]AvailableTaskRef, 0)
	for _, jobID := range cfg.Jobs.Keys {
		jobVal, ok := cfg.Jobs.Get(jobID)
		if !ok {
			continue
		}
		job, ok := AsJob(jobVal)
		if !ok {
			continue
		}
		wlJob, ok := byJob[jobID]
		if !ok {
			continue
		}
		ordered := OrderedTaskIDs(job.Tasks)
		if wlJob.CanStart && len(ordered) > 0 {
			out = append(out, AvailableTaskRef{
				JobID:         jobID,
				TaskID:        ordered[0],
				JobInstanceID: "",
			})
		}
		for _, inst := range wlJob.Instances {
			for _, taskID := range ordered {
				if inst.Tasks[taskID] == TaskStatusAvailable {
					out = append(out, AvailableTaskRef{
						JobID:         jobID,
						TaskID:        taskID,
						JobInstanceID: inst.ID,
					})
				}
			}
		}
	}
	return out
}

// FirstAvailableTask returns the first CollectAvailableTaskRefs entry.
func FirstAvailableTask(cfg OrgConfig, wl Worklist) (AvailableTaskRef, bool) {
	refs := CollectAvailableTaskRefs(cfg, wl)
	if len(refs) == 0 {
		return AvailableTaskRef{}, false
	}
	return refs[0], true
}

// BuildTaskFormBundle projects schema/ui + chrome/derived rules for one Task.
// Living code stays generic (no plant Task id branches).
func BuildTaskFormBundle(cfg OrgConfig, domain Domain, ref AvailableTaskRef) (*TaskFormBundle, error) {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return nil, fmt.Errorf("living: unknown job %q", ref.JobID)
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil, fmt.Errorf("living: invalid job %q", ref.JobID)
	}
	task, ok := TaskByID(job.Tasks, ref.TaskID)
	if !ok {
		return nil, fmt.Errorf("living: unknown task %s/%s", ref.JobID, ref.TaskID)
	}
	payload := taskPayload(task)
	if payload == nil {
		return nil, fmt.Errorf("living: empty task payload %s/%s", ref.JobID, ref.TaskID)
	}

	projected, err := projectTaskForm(domain, payload, ref.JobInstanceID, jobInstanceLabel(domain, ref.JobID, ref.JobInstanceID))
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

	derived := compileDerivedRules(payload, projected.LeftoverIDs)
	surface := "live"
	if isLeftoverTablePayload(payload) {
		surface = "leftover-table"
	}

	sealFieldIDs := sortedKeys(sealIDs)
	title := taskTitle(payload, ref.TaskID)
	projectionKey := ""
	if isMapTablePayload(payload) && len(projected.LeftoverIDs) > 0 {
		projectionKey = leftoversIdentity(projected.LeftoverIDs)
	}

	initial := seedInitialData(domain, payload, projected)
	bindings := buildFormBindings(domain, payload, ref)

	bundle := &TaskFormBundle{
		JobID:          ref.JobID,
		TaskID:         ref.TaskID,
		JobInstanceID:  ref.JobInstanceID,
		Title:          title,
		Surface:        surface,
		Schema:         schema,
		UiSchema:       uiSchema,
		InitialData:    initial,
		LeftoverIDs:    projected.LeftoverIDs,
		Columns:        projected.Columns,
		Chrome:         chrome,
		Derived:        derived,
		ChromeFieldIDs: chromeFieldIDs,
		SealFieldIDs:   sealFieldIDs,
		ProjectionKey:  projectionKey,
		Bindings:       bindings,
	}
	return bundle, nil
}

// ProjectAvailableBundles builds Task form bundles for every CollectAvailableTaskRefs
// address (available + provisional). Empty when none.
func ProjectAvailableBundles(cfg OrgConfig, domain Domain, wl Worklist) ([]*TaskFormBundle, error) {
	refs := CollectAvailableTaskRefs(cfg, wl)
	out := make([]*TaskFormBundle, 0, len(refs))
	for _, ref := range refs {
		bundle, err := BuildTaskFormBundle(cfg, domain, ref)
		if err != nil {
			return nil, err
		}
		out = append(out, bundle)
	}
	return out, nil
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

func buildFormBindings(domain Domain, payload map[string]any, ref AvailableTaskRef) map[string]any {
	// DomainSnapshot-shaped bag for SPA Chrome (finalResults needs submissions).
	subs := make([]any, len(domain.Submissions))
	for i, s := range domain.Submissions {
		subs[i] = s
	}
	domainBag := map[string]any{
		"load":        entityMapToAny(domain.Load),
		"pallet":      entityMapToAny(domain.Pallet),
		"container":   entityMapToAny(domain.Container),
		"submissions": subs,
	}
	if tare, ok := payload["tareKinds"]; ok {
		domainBag["tareKinds"] = deepCloneJSON(tare)
	}
	bindings := map[string]any{"domain": domainBag}
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
	primary, _, ok := primaryDynamicProperty(payload)
	if !ok || len(projected.LeftoverIDs) == 0 {
		return out
	}
	if isLeftoverTablePayload(payload) {
		progress := map[string]any{}
		for _, id := range projected.LeftoverIDs {
			facts, hit := domain.Pallet[id]
			if !hit {
				continue
			}
			p, _ := facts["progress"].(string)
			if p == "partial" || p == "done" {
				progress[id] = p
			}
		}
		if len(progress) > 0 {
			out[primary] = progress
		}
		return out
	}
	if isMapTablePayload(payload) {
		rows := map[string]any{}
		for _, id := range projected.LeftoverIDs {
			rows[id] = map[string]any{
				"weight":     "",
				"tareKind":   "euro",
				"idoneita":   "NIR",
				"panelCount": 0,
			}
		}
		out[primary] = rows
		out["tareKind"] = "euro"
		return out
	}
	return out
}

func leftoversIdentity(ids []string) string {
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
