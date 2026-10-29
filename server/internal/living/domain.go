package living

// Domain is the org's accumulating facts (entities + submissions + job instances).
type Domain struct {
	Load         EntityMap     `json:"load"`
	Pallet       EntityMap     `json:"pallet"`
	Container    EntityMap     `json:"container"`
	Submissions  []Submission  `json:"submissions"`
	JobInstances []JobInstance `json:"jobInstances"`
}

// EntityMap is id → facts for one entity type.
type EntityMap map[string]map[string]any

// Submission records a successful Task Commit on a Job instance.
type Submission struct {
	JobID        string         `json:"jobId"`
	TaskID       string         `json:"taskId"`
	JobInstanceID string        `json:"jobInstanceId,omitempty"`
	Timestamp    string         `json:"timestamp"`
	UserID       string         `json:"userId"`
	Facts        map[string]any `json:"facts"`
	IDsTouched   []string       `json:"idsTouched"`
}

// JobInstance is one occurrence of a repeatable Job in the Domain.
type JobInstance struct {
	JobID string `json:"jobId"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

// EmptyDomain returns a Domain with no entities, submissions, or job instances.
func EmptyDomain() Domain {
	return Domain{
		Load:         EntityMap{},
		Pallet:       EntityMap{},
		Container:    EntityMap{},
		Submissions:  []Submission{},
		JobInstances: []JobInstance{},
	}
}
