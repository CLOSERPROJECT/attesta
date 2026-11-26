package org

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// InstancePassportDTO is one issued Passport for the Org instance board aside.
type InstancePassportDTO struct {
	SubjectID   string `json:"subjectId"`
	Label       string `json:"label"`
	IssuedAt    string `json:"issuedAt,omitempty"`
	DigitalLink string `json:"digitalLink"`
}

// InstancePassportsView is GET /api/org/passports for one stream instance.
type InstancePassportsView struct {
	Passports []InstancePassportDTO `json:"passports"`
}

// CollectInstancePassports lists Passports for subjects whose onSubmit.passport
// Task is done on a Job instance belonging to focus. Config-driven: no plant
// entity branches. Boot focus or missing passport block → empty.
func CollectInstancePassports(
	ctx context.Context,
	cfg OrgConfig,
	domain Domain,
	focus TimelineFocus,
	store DPPStore,
) ([]InstancePassportDTO, error) {
	if store == nil || cfg.DPP == nil || focus.IsBoot() {
		return nil, nil
	}
	instanceID, ok := focus.InstanceID()
	if !ok {
		return nil, nil
	}
	scope, err := ScopeFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]InstancePassportDTO, 0)
	for _, ref := range passportOptInTaskRefs(cfg) {
		for _, ji := range domain.JobInstances {
			if ji.JobID != ref.JobID {
				continue
			}
			subjectID := strings.TrimSpace(ji.ID)
			if subjectID == "" {
				continue
			}
			if !JobInstanceBelongsToStream(cfg, domain, ref.JobID, subjectID, instanceID) {
				continue
			}
			if !EvaluatePathAvailability(cfg, domain, ref, subjectID).Done {
				continue
			}
			if _, dup := seen[subjectID]; dup {
				continue
			}
			rec, getErr := store.Get(ctx, scope, subjectID)
			if errors.Is(getErr, ErrNotFound) {
				continue
			}
			if getErr != nil {
				return nil, getErr
			}
			seen[subjectID] = struct{}{}
			out = append(out, instancePassportDTO(rec))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IssuedAt != out[j].IssuedAt {
			return out[i].IssuedAt > out[j].IssuedAt
		}
		return out[i].SubjectID < out[j].SubjectID
	})
	return out, nil
}

func passportOptInTaskRefs(cfg OrgConfig) []TaskPathRef {
	var out []TaskPathRef
	for _, jobID := range cfg.Jobs.Keys {
		jobVal, ok := cfg.Jobs.Get(jobID)
		if !ok {
			continue
		}
		job, ok := AsJob(jobVal)
		if !ok {
			continue
		}
		for _, taskID := range OrderedTaskIDs(job.Tasks) {
			tc, ok := TaskByID(job.Tasks, taskID)
			if !ok || !taskOnSubmitDPP(map[string]any(tc)) {
				continue
			}
			out = append(out, TaskPathRef{JobID: jobID, TaskID: taskID})
		}
	}
	return out
}

func instancePassportDTO(rec DPPRecord) InstancePassportDTO {
	label := strings.TrimSpace(rec.ProductName)
	if label == "" {
		label = strings.TrimSpace(rec.SubjectID)
	}
	return InstancePassportDTO{
		SubjectID:   strings.TrimSpace(rec.SubjectID),
		Label:       label,
		IssuedAt:    strings.TrimSpace(rec.IssuedAt),
		DigitalLink: strings.TrimSpace(rec.DigitalLink),
	}
}
