package living

// Domain is the org's accumulating facts (entities + submissions + job instances).
// Well-known kinds stay on Load/Pallet/Container for wire compatibility; Extra
// holds any additional kinds declared in Org Config domain.entities.
type Domain struct {
	Load         EntityMap             `json:"load"`
	Pallet       EntityMap             `json:"pallet"`
	Container    EntityMap             `json:"container"`
	Extra        map[string]*EntityMap `json:"extra,omitempty"`
	Submissions  []Submission          `json:"submissions"`
	JobInstances []JobInstance         `json:"jobInstances"`
}

// EntityMap is id → facts for one entity type.
type EntityMap map[string]map[string]any

// Submission records a successful Task Commit on a Job instance.
type Submission struct {
	JobID         string         `json:"jobId"`
	TaskID        string         `json:"taskId"`
	JobInstanceID string         `json:"jobInstanceId,omitempty"`
	Timestamp     string         `json:"timestamp"`
	UserID        string         `json:"userId"`
	Facts         map[string]any `json:"facts"`
	IDsTouched    []string       `json:"idsTouched"`
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
		Extra:        nil,
		Submissions:  []Submission{},
		JobInstances: []JobInstance{},
	}
}

// EntityMaps returns kind → facts for transform / read bindings.
func (d Domain) EntityMaps() map[string]EntityMap {
	out := map[string]EntityMap{
		"load":      d.Load,
		"pallet":    d.Pallet,
		"container": d.Container,
	}
	for kind, m := range d.Extra {
		if m == nil {
			out[kind] = EntityMap{}
			continue
		}
		out[kind] = *m
	}
	return out
}
