package living

// Domain is the org's accumulating facts (entities + submissions + job instances).
// Entities kind names are Org Config vocabulary (e.g. load); EntityMaps shares
// nested map headers with Entities (caller must clone before mutating).
type Domain struct {
	Entities     map[string]EntityMap `json:"entities"`
	Submissions  []Submission         `json:"submissions"`
	JobInstances []JobInstance        `json:"jobInstances"`
}

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

const (
	kindLoad      = "load"
	kindPallet    = "pallet"
	kindContainer = "container"
)

func EmptyDomain() Domain {
	return Domain{
		Entities:     map[string]EntityMap{},
		Submissions:  []Submission{},
		JobInstances: []JobInstance{},
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
