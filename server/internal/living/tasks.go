package living

import "fmt"

// TaskConfig is one authored Task under a Job. Wire JSON/YAML require unique "id";
// remaining keys are the payload (roles, schema, onSubmit, …).
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
	seen := make(map[string]struct{}, len(tasks))
	for i, t := range tasks {
		id := t.ID()
		if id == "" {
			return fmt.Errorf("tasks[%d]: id required", i)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("tasks: duplicate id %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
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
