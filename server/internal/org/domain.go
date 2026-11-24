package org

// Domain is the org's accumulating facts (entities + submissions + job instances).
// Entities kind names are Org Config vocabulary (e.g. load); EntityMaps shares
// nested map headers with Entities (caller must clone before mutating).
type Domain struct {
	Entities     map[string]EntityMap `json:"entities"`
	Submissions  []Submission         `json:"submissions"`
	JobInstances []JobInstance        `json:"jobInstances"`
}

type EntityMap map[string]map[string]any

// EntityTouch is one Domain entity this Submission minted or patched.
type EntityTouch struct {
	Op     string `json:"op"`     // "mint" | "patch"
	Entity string `json:"entity"` // kind, e.g. "lot"
	ID     string `json:"id"`
}

// Submission records a successful Task Commit on a Job instance.
type Submission struct {
	JobID           string         `json:"jobId"`
	TaskID          string         `json:"taskId"`
	JobInstanceID   string         `json:"jobInstanceId,omitempty"`
	Timestamp       string         `json:"timestamp"`
	UserID          string         `json:"userId"`
	// UserName is the human-facing actor label at Commit (email / display name).
	UserName        string         `json:"userName,omitempty"`
	Facts           map[string]any `json:"facts"`
	EntitiesTouched []EntityTouch  `json:"entitiesTouched"`
}

// JobInstance is one occurrence of a repeatable Job in the Domain.
type JobInstance struct {
	JobID string `json:"jobId"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

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
