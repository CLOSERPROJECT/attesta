package living

// OrgConfig is one Config version of the org's authored operating picture.
type OrgConfig struct {
	ID      string         `json:"id"`
	Org     string         `json:"org,omitempty"`
	Version string         `json:"version"`
	Domain  map[string]any `json:"domain,omitempty"`
	Roles   []string       `json:"roles,omitempty"`
	Jobs    OrderedObject  `json:"jobs"`
}

// Job is a top-level authored chapter (intake, line, …).
type Job struct {
	Repeatable bool          `json:"repeatable"`
	InstanceID string        `json:"instanceId"`
	Label      string        `json:"label"`
	StartLabel string        `json:"startLabel,omitempty"`
	StartsWhen *StartsWhen   `json:"startsWhen,omitempty"`
	Tasks      OrderedObject `json:"tasks"`
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
