package living

import "fmt"

// TaskConfig is one authored Task or Task group under a Job. Wire JSON/YAML
// require unique "id"; remaining keys are the payload (roles, schema, onSubmit,
// or for groups: kind, children, …).
type TaskConfig map[string]any

// ID returns the authored Task id, or "" if missing/invalid.
func (t TaskConfig) ID() string {
	if t == nil {
		return ""
	}
	id, _ := t["id"].(string)
	return id
}

func (t TaskConfig) IsGroup() bool {
	if t == nil {
		return false
	}
	kind, _ := t["kind"].(string)
	return kind == "group"
}

// ChildTasks returns ordered children from JSON "tasks" or YAML "substeps".
func (t TaskConfig) ChildTasks() []TaskConfig {
	if t == nil {
		return nil
	}
	raw, ok := t["tasks"]
	if !ok {
		raw, ok = t["substeps"]
	}
	if !ok || raw == nil {
		return nil
	}
	return decodeTaskConfigList(raw)
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

// LeafTaskByPath finds the leaf TaskConfig addressed by ref under job tasks.
// Flat refs match a top-level leaf; nested refs walk one group level.
func LeafTaskByPath(tasks []TaskConfig, ref TaskPathRef) (TaskConfig, bool) {
	if ref.Nested() {
		group, ok := TaskByID(tasks, ref.GroupID)
		if !ok || !group.IsGroup() {
			return nil, false
		}
		leaf, ok := TaskByID(group.ChildTasks(), ref.TaskID)
		if !ok || leaf.IsGroup() {
			return nil, false
		}
		return leaf, true
	}
	leaf, ok := TaskByID(tasks, ref.TaskID)
	if !ok || leaf.IsGroup() {
		return nil, false
	}
	return leaf, true
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

		children := t.ChildTasks()
		if t.IsGroup() {
			if len(children) == 0 {
				return fmt.Errorf("%s %q: group requires non-empty children", prefix, id)
			}
			if hasLeafCommitFields(t) {
				return fmt.Errorf("%s %q: group must not declare leaf Commit fields (schema/onSubmit)", prefix, id)
			}
			for _, child := range children {
				if child.IsGroup() {
					return fmt.Errorf("%s %q: group children must be leaves (one nesting level)", prefix, id)
				}
			}
			if err := validateTaskConfigsAt(children, prefix+".tasks"); err != nil {
				return err
			}
			continue
		}
		if len(children) > 0 {
			return fmt.Errorf("%s %q: leaf must not declare nested tasks (set kind: group)", prefix, id)
		}
	}
	return nil
}

func hasLeafCommitFields(t TaskConfig) bool {
	if t == nil {
		return false
	}
	if _, ok := t["schema"]; ok {
		return true
	}
	if _, ok := t["onSubmit"]; ok {
		return true
	}
	return false
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
