package living

import (
	"fmt"
	"strings"
)

type EntityType string

const (
	EntityLoad      EntityType = "load"
	EntityPallet    EntityType = "pallet"
	EntityContainer EntityType = "container"
)

type MintOp struct {
	Entity EntityType
	ID     string
	Set    map[string]any
}

// PatchOp merges facts; null values unset keys.
type PatchOp struct {
	Entity EntityType
	ID     string
	Set    map[string]any
}

type MutateOp interface {
	mutateOp()
}

func (MintOp) mutateOp()  {}
func (PatchOp) mutateOp() {}

type MutateRequest struct {
	JobID            string
	TaskID           string
	JobInstanceID    string
	Facts            map[string]any
	Ops              []MutateOp
	Timestamp        string
	UserID           string
	AppendSubmission *bool
	NewJobInstance   *JobInstance
}

type MutateError struct {
	Kind   string
	Entity EntityType
	ID     string
}

func (e MutateError) Error() string {
	return fmt.Sprintf("living: mutate %s %s/%s", e.Kind, e.Entity, e.ID)
}

func Mutate(domain Domain, req MutateRequest) (Domain, error) {
	next := cloneDomain(domain)
	idsTouched := make([]string, 0, len(req.Ops))

	for _, op := range req.Ops {
		switch o := op.(type) {
		case MintOp:
			m, err := entityMap(&next, o.Entity)
			if err != nil {
				return Domain{}, err
			}
			if _, exists := (*m)[o.ID]; exists {
				return Domain{}, MutateError{Kind: "duplicate-id", Entity: o.Entity, ID: o.ID}
			}
			(*m)[o.ID] = cloneFacts(o.Set)
			idsTouched = append(idsTouched, o.ID)
		case PatchOp:
			m, err := entityMap(&next, o.Entity)
			if err != nil {
				return Domain{}, err
			}
			existing, exists := (*m)[o.ID]
			if !exists {
				return Domain{}, MutateError{Kind: "missing-id", Entity: o.Entity, ID: o.ID}
			}
			(*m)[o.ID] = applyFactsPatch(existing, o.Set)
			idsTouched = append(idsTouched, o.ID)
		default:
			return Domain{}, fmt.Errorf("living: unknown mutate op %T", op)
		}
	}

	appendSubmission := true
	if req.AppendSubmission != nil {
		appendSubmission = *req.AppendSubmission
	}
	if appendSubmission {
		if strings.TrimSpace(req.UserID) == "" {
			return Domain{}, fmt.Errorf("living: userId required when appending a Submission")
		}
		sub := Submission{
			JobID:         req.JobID,
			TaskID:        req.TaskID,
			JobInstanceID: req.JobInstanceID,
			Timestamp:     req.Timestamp,
			UserID:        req.UserID,
			Facts:         cloneFacts(req.Facts),
			IDsTouched:    idsTouched,
		}
		next.Submissions = append(next.Submissions, sub)
	}

	if req.NewJobInstance != nil {
		next.JobInstances = append(next.JobInstances, *req.NewJobInstance)
	}

	return next, nil
}

func entityMap(d *Domain, entity EntityType) (*EntityMap, error) {
	switch entity {
	case EntityLoad:
		if d.Load == nil {
			d.Load = EntityMap{}
		}
		return &d.Load, nil
	case EntityPallet:
		if d.Pallet == nil {
			d.Pallet = EntityMap{}
		}
		return &d.Pallet, nil
	case EntityContainer:
		if d.Container == nil {
			d.Container = EntityMap{}
		}
		return &d.Container, nil
	default:
		return nil, fmt.Errorf("living: unknown entity %q", entity)
	}
}

func cloneFacts(facts map[string]any) map[string]any {
	if facts == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(facts))
	for k, v := range facts {
		out[k] = v
	}
	return out
}

func applyFactsPatch(existing map[string]any, set map[string]any) map[string]any {
	next := cloneFacts(existing)
	for k, v := range set {
		if v == nil {
			delete(next, k)
			continue
		}
		next[k] = v
	}
	return next
}
