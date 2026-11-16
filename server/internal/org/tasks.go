package org

import "fmt"

// TaskConfig is one authored leaf Task under a Job. Wire JSON/YAML require
// unique "id"; remaining keys are the payload (roles, schema, onSubmit, …).
type TaskConfig map[string]any

// ID returns the authored Task id, or "" if missing/invalid.
func (t TaskConfig) ID() string {
	if t == nil {
		return ""
	}
	id, _ := t["id"].(string)
	return id
}

// OrderedTaskIDs returns Task ids in authored (Unlock) order.
func OrderedTaskIDs(tasks []TaskConfig) []string {
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, t.ID())
	}
	return out
}

// TaskByID finds a Task by id in authored order.
func TaskByID(tasks []TaskConfig, id string) (TaskConfig, bool) {
	for _, t := range tasks {
		if t.ID() == id {
			return t, true
		}
	}
	return nil, false
}

func validateTaskConfigs(tasks []TaskConfig) error {
	return validateTaskConfigsAt(tasks, "tasks")
}

func validateTaskConfigsAt(tasks []TaskConfig, path string) error {
	seen := make(map[string]struct{}, len(tasks))
	for i, t := range tasks {
		id := t.ID()
		prefix := fmt.Sprintf("%s[%d]", path, i)
		if id == "" {
			return fmt.Errorf("%s: id required", prefix)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%s: duplicate id %q", path, id)
		}
		seen[id] = struct{}{}

		if err := rejectGroupAuthoring(t, prefix); err != nil {
			return err
		}
		if err := validateUnlockFrom(tasks, i, t, prefix); err != nil {
			return err
		}
		if err := validateTaskSubmit(t, prefix); err != nil {
			return err
		}
	}
	return nil
}

// rejectGroupAuthoring fails closed on kind:group and group-only keys.
func rejectGroupAuthoring(t TaskConfig, prefix string) error {
	if t == nil {
		return nil
	}
	id := t.ID()
	if kind, _ := t["kind"].(string); kind == "group" {
		return fmt.Errorf("%s %q: kind %q is not supported", prefix, id, kind)
	}
	for _, key := range []string{
		"closeRequiresChildSubmission",
		"closeLabel",
		"instanceLabelEntity",
	} {
		if _, ok := t[key]; ok {
			return fmt.Errorf("%s %q: %s is not supported", prefix, id, key)
		}
	}
	if _, ok := t["tasks"]; ok {
		return fmt.Errorf("%s %q: nested tasks are not supported", prefix, id)
	}
	if _, ok := t["substeps"]; ok {
		return fmt.Errorf("%s %q: nested tasks are not supported", prefix, id)
	}
	return nil
}

const (
	submitOnce = "once"
	submitMany = "many"
)

// taskSubmit returns authored submit cardinality: once (default) or many.
func taskSubmit(t TaskConfig) string {
	if t == nil {
		return submitOnce
	}
	raw, ok := t["submit"]
	if !ok || raw == nil {
		return submitOnce
	}
	s, ok := raw.(string)
	if !ok {
		return submitOnce
	}
	switch s {
	case submitMany:
		return submitMany
	default:
		return submitOnce
	}
}

func taskSubmitMany(t TaskConfig) bool {
	return taskSubmit(t) == submitMany
}

func validateTaskSubmit(t TaskConfig, prefix string) error {
	raw, present := t["submit"]
	if !present {
		return nil
	}
	id := t.ID()
	s, ok := raw.(string)
	if !ok {
		return fmt.Errorf("%s %q: submit must be %q or %q", prefix, id, submitOnce, submitMany)
	}
	switch s {
	case submitOnce, submitMany:
		return nil
	default:
		return fmt.Errorf("%s %q: submit must be %q or %q", prefix, id, submitOnce, submitMany)
	}
}

// validateUnlockFrom: optional unlockFrom must name a strictly earlier sibling
// in the same list (no first-task, empty, non-string, self, forward, or unknown).
func validateUnlockFrom(tasks []TaskConfig, index int, t TaskConfig, prefix string) error {
	raw, present := t["unlockFrom"]
	if !present {
		return nil
	}
	id := t.ID()
	if index == 0 {
		return fmt.Errorf("%s %q: first task must not declare unlockFrom", prefix, id)
	}
	from, ok := raw.(string)
	if !ok {
		return fmt.Errorf("%s %q: unlockFrom must be a string task id", prefix, id)
	}
	if from == "" {
		return fmt.Errorf("%s %q: unlockFrom must be a non-empty task id", prefix, id)
	}
	if from == id {
		return fmt.Errorf("%s %q: unlockFrom must not reference self", prefix, id)
	}
	for j := 0; j < index; j++ {
		if tasks[j].ID() == from {
			return nil
		}
	}
	for j := index + 1; j < len(tasks); j++ {
		if tasks[j].ID() == from {
			return fmt.Errorf("%s %q: unlockFrom %q must name an earlier sibling", prefix, id, from)
		}
	}
	return fmt.Errorf("%s %q: unlockFrom %q is not a sibling task id", prefix, id, from)
}

const (
	commitNavAdvance  = "advance"
	commitNavHome     = "home"
	commitNavOverview = "overview"
	commitNavStay     = "stay"
)

func taskOnCommit(task map[string]any) map[string]any {
	if task == nil {
		return nil
	}
	on, _ := task["onCommit"].(map[string]any)
	return on
}

// taskCommitNavigation reads onCommit.navigation; omit or unknown ⇒ advance.
func taskCommitNavigation(task map[string]any) string {
	on := taskOnCommit(task)
	if on == nil {
		return commitNavAdvance
	}
	nav, _ := on["navigation"].(string)
	switch nav {
	case commitNavHome, commitNavOverview, commitNavStay, commitNavAdvance:
		return nav
	default:
		return commitNavAdvance
	}
}

// taskAdvanceTarget reads onCommit.advanceTarget; omit ⇒ true.
func taskAdvanceTarget(task map[string]any) bool {
	on := taskOnCommit(task)
	if on == nil {
		return true
	}
	raw, ok := on["advanceTarget"]
	if !ok {
		return true
	}
	b, ok := raw.(bool)
	if !ok {
		return true
	}
	return b
}

func decodeTaskConfigList(raw any) []TaskConfig {
	switch list := raw.(type) {
	case []TaskConfig:
		out := make([]TaskConfig, len(list))
		copy(out, list)
		return out
	case []any:
		out := make([]TaskConfig, 0, len(list))
		for _, item := range list {
			if tc := asTaskConfig(item); tc != nil {
				out = append(out, tc)
			}
		}
		return out
	default:
		return nil
	}
}

func asTaskConfig(v any) TaskConfig {
	switch t := v.(type) {
	case TaskConfig:
		return t
	case map[string]any:
		return TaskConfig(t)
	default:
		return nil
	}
}

func taskPayload(task any) map[string]any {
	switch t := task.(type) {
	case TaskConfig:
		return map[string]any(t)
	case map[string]any:
		return t
	default:
		return nil
	}
}
