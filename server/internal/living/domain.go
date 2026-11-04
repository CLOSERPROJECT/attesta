package living

// Domain is the org's accumulating facts (entities + submissions + job instances).
// Entities kind names are Org Config vocabulary (e.g. load); EntityMaps shares
// nested map headers with Entities (caller must clone before mutating).
type Domain struct {
	Entities       map[string]EntityMap `json:"entities"`
	Submissions    []Submission         `json:"submissions"`
	JobInstances   []JobInstance        `json:"jobInstances"`
	GroupInstances []GroupInstance      `json:"groupInstances"`
	ClosedGroups   []ClosedGroup        `json:"closedGroups"`
}

type EntityMap map[string]map[string]any

// Submission records a successful Task Commit on a Job instance.
// Nested leaf Commits set GroupID + GroupInstanceID (ADR-0006 path).
type Submission struct {
	JobID           string         `json:"jobId"`
	TaskID          string         `json:"taskId"`
	JobInstanceID   string         `json:"jobInstanceId,omitempty"`
	GroupID         string         `json:"groupId,omitempty"`
	GroupInstanceID string         `json:"groupInstanceId,omitempty"`
	Timestamp       string         `json:"timestamp"`
	UserID          string         `json:"userId"`
	Facts           map[string]any `json:"facts"`
	IDsTouched      []string       `json:"idsTouched"`
}

// JobInstance is one occurrence of a repeatable Job in the Domain.
type JobInstance struct {
	JobID string `json:"jobId"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

// GroupInstance is one occurrence of a Task group under a Job instance.
type GroupInstance struct {
	JobID         string `json:"jobId"`
	JobInstanceID string `json:"jobInstanceId"`
	GroupID       string `json:"groupId"`
	ID            string `json:"id"`
}

// ClosedGroup records that a Task group was closed on a Job instance.
// Unlock of the next top-level sibling reads this — not child Submissions.
type ClosedGroup struct {
	JobID         string `json:"jobId"`
	JobInstanceID string `json:"jobInstanceId"`
	GroupID       string `json:"groupId"`
}

func EmptyDomain() Domain {
	return Domain{
		Entities:       map[string]EntityMap{},
		Submissions:    []Submission{},
		JobInstances:   []JobInstance{},
		GroupInstances: []GroupInstance{},
		ClosedGroups:   []ClosedGroup{},
	}
}

func (d Domain) EntityMaps() map[string]EntityMap {
	ensureEntities(&d)
	out := make(map[string]EntityMap, len(d.Entities))
	for kind, m := range d.Entities {
		out[kind] = nonNilEntityMap(m)
	}
	return out
}

func (d Domain) Kind(kind string) EntityMap {
	ensureEntities(&d)
	return nonNilEntityMap(d.Entities[kind])
}

func domainFromKinds(kinds map[string]EntityMap) Domain {
	d := EmptyDomain()
	for kind, m := range kinds {
		d.Entities[kind] = nonNilEntityMap(m)
	}
	return d
}

func nonNilEntityMap(m EntityMap) EntityMap {
	if m == nil {
		return EntityMap{}
	}
	return m
}

func putKind(d *Domain, kind string, m EntityMap) {
	ensureEntities(d)
	d.Entities[kind] = nonNilEntityMap(m)
}

func ensureEntities(d *Domain) {
	if d.Entities == nil {
		d.Entities = map[string]EntityMap{}
	}
}
