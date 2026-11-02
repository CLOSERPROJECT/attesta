package living

import (
	"encoding/json"
)

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

// UnmarshalJSON prefers entities; folds legacy load/pallet/container/extra when absent.
func (d *Domain) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := EmptyDomain()
	if v, ok := raw["entities"]; ok {
		var entities map[string]EntityMap
		if err := json.Unmarshal(v, &entities); err != nil {
			return err
		}
		for kind, m := range entities {
			out.Entities[kind] = nonNilEntityMap(m)
		}
	}
	ingestLegacy := func(key string) error {
		v, ok := raw[key]
		if !ok {
			return nil
		}
		var m EntityMap
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		if _, exists := out.Entities[key]; !exists {
			out.Entities[key] = nonNilEntityMap(m)
		}
		return nil
	}
	for _, key := range []string{kindLoad, kindPallet, kindContainer} {
		if err := ingestLegacy(key); err != nil {
			return err
		}
	}
	if v, ok := raw["extra"]; ok {
		var extra map[string]EntityMap
		if err := json.Unmarshal(v, &extra); err != nil {
			return err
		}
		for kind, m := range extra {
			if _, exists := out.Entities[kind]; exists {
				continue
			}
			out.Entities[kind] = nonNilEntityMap(m)
		}
	}
	if v, ok := raw["submissions"]; ok {
		if err := json.Unmarshal(v, &out.Submissions); err != nil {
			return err
		}
	}
	if v, ok := raw["jobInstances"]; ok {
		if err := json.Unmarshal(v, &out.JobInstances); err != nil {
			return err
		}
	}
	if out.Submissions == nil {
		out.Submissions = []Submission{}
	}
	if out.JobInstances == nil {
		out.JobInstances = []JobInstance{}
	}
	*d = out
	return nil
}
