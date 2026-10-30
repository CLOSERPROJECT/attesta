package living

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// OrgConfig is one Config version of the org's authored operating picture.
type OrgConfig struct {
	ID      string         `json:"id"`
	Org     string         `json:"org,omitempty"`
	Version string         `json:"version"`
	Domain  map[string]any `json:"domain,omitempty"`
	Roles   []string       `json:"roles,omitempty"`
	Jobs    OrderedObject  `json:"jobs"`
}

// UnmarshalJSON decodes each Job from raw JSON so Jobs OrderedObject keeps
// authored key order and nested Tasks stay typed ([]TaskConfig).
func (c *OrgConfig) UnmarshalJSON(data []byte) error {
	var wire struct {
		ID      string          `json:"id"`
		Org     string          `json:"org,omitempty"`
		Version string          `json:"version"`
		Domain  map[string]any  `json:"domain,omitempty"`
		Roles   []string        `json:"roles,omitempty"`
		Jobs    json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	jobs, err := decodeOrderedJobs(wire.Jobs)
	if err != nil {
		return err
	}
	c.ID = wire.ID
	c.Org = wire.Org
	c.Version = wire.Version
	c.Domain = wire.Domain
	c.Roles = wire.Roles
	c.Jobs = jobs
	return nil
}

func decodeOrderedJobs(data json.RawMessage) (OrderedObject, error) {
	if len(data) == 0 || string(data) == "null" {
		return OrderedObject{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return OrderedObject{}, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return OrderedObject{}, fmt.Errorf("org config jobs: expected '{', got %v", tok)
	}
	out := newOrderedObject()
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return OrderedObject{}, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return OrderedObject{}, fmt.Errorf("org config jobs: expected string key, got %T", keyTok)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return OrderedObject{}, err
		}
		var job Job
		if err := json.Unmarshal(raw, &job); err != nil {
			return OrderedObject{}, fmt.Errorf("org config job %q: %w", key, err)
		}
		out.Set(key, job)
	}
	tok, err = dec.Token()
	if err != nil {
		return OrderedObject{}, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '}' {
		return OrderedObject{}, fmt.Errorf("org config jobs: expected '}', got %v", tok)
	}
	return out, nil
}

// AsJob returns a typed Job. OrgConfig JSON decode stores Job; callers still
// coerce in case a store left a raw message.
func AsJob(v any) (Job, bool) {
	if v == nil {
		return Job{}, false
	}
	if j, ok := v.(Job); ok {
		return j, true
	}
	if raw, ok := v.(json.RawMessage); ok {
		var out Job
		if err := json.Unmarshal(raw, &out); err != nil {
			return Job{}, false
		}
		return out, true
	}
	return Job{}, false
}

// normalizeOrgConfigJobs reifies any leftover non-Job values after Get.
// OrgConfig.UnmarshalJSON already stores typed Job; this covers RawMessage.
func normalizeOrgConfigJobs(cfg *OrgConfig) {
	if cfg == nil || cfg.Jobs.Values == nil {
		return
	}
	for _, key := range cfg.Jobs.Keys {
		v, ok := cfg.Jobs.Values[key]
		if !ok {
			continue
		}
		if _, ok := v.(Job); ok {
			continue
		}
		if job, ok := AsJob(v); ok {
			cfg.Jobs.Values[key] = job
		}
	}
}

// Job is a top-level authored chapter (intake, line, …).
type Job struct {
	Repeatable bool         `json:"repeatable"`
	InstanceID string       `json:"instanceId"`
	Label      string       `json:"label"`
	StartLabel string       `json:"startLabel,omitempty"`
	StartsWhen *StartsWhen  `json:"startsWhen,omitempty"`
	Tasks      []TaskConfig `json:"tasks"`
}

// UnmarshalJSON validates unique non-empty Task ids after decode.
func (j *Job) UnmarshalJSON(data []byte) error {
	type wire Job
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	if err := validateTaskConfigs(w.Tasks); err != nil {
		return err
	}
	*j = Job(w)
	return nil
}

// StartsWhen is a Job-level start gate over Submissions of a named Task path.
type StartsWhen struct {
	Task       StartsWhenRef `json:"task"`
	Quantifier string        `json:"quantifier"`
}

// StartsWhenRef addresses a Task as jobId + taskId (Task path job/task).
type StartsWhenRef struct {
	JobID  string `json:"jobId"`
	TaskID string `json:"taskId"`
}
