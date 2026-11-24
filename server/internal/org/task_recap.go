package org

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// TaskRecap is one Submission row for overview expand: facts + actor/time,
// plus any post-Commit links side record. No schema, uiSchema, or Domain —
// only optional property titles from Active Org Config.
type TaskRecap struct {
	JobID         string         `json:"jobId"`
	JobInstanceID string         `json:"jobInstanceId"`
	TaskID        string         `json:"taskId"`
	Timestamp     string         `json:"timestamp"`
	UserID        string         `json:"userId"`
	// UserName is the human-facing actor; falls back to UserID when unset (legacy rows).
	UserName      string         `json:"userName"`
	Facts         map[string]any `json:"facts"`
	// Titles maps authored property keys → schema title (Italian plant copy).
	// Omitted keys fall back to the raw fact key in the SPA.
	Titles map[string]string `json:"titles,omitempty"`
	Links  []HttpLinkItem    `json:"links,omitempty"`
}

// TaskRecapList is the task-recap wire: always an items array (once → 0–1,
// many → N), oldest timestamp first.
type TaskRecapList struct {
	Items []TaskRecap `json:"items"`
}

// LoadTaskRecap returns all Domain Submissions at addr (oldest first), each
// with facts + LinkStore links keyed by that row's timestamp. Auth is Caller's
// concern (Load-level Gate + org); this method does not Cerbos-authorize.
// Unknown address or no matching Submissions → ErrNotFound.
func (l *DomainLoader) LoadTaskRecap(
	ctx context.Context,
	scope Scope,
	addr SubmissionAddress,
) (TaskRecapList, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return TaskRecapList{}, err
	}
	addr = addr.normalized()
	if !addr.valid() {
		return TaskRecapList{}, fmt.Errorf("org: submission address required")
	}
	if l.Domains == nil {
		return TaskRecapList{}, fmt.Errorf("org: stores required")
	}

	domain, err := l.Domains.Get(ctx, scope)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return TaskRecapList{}, ErrNotFound
		}
		return TaskRecapList{}, err
	}

	matched := submissionsMatching(domain.Submissions, addr)
	if len(matched) == 0 {
		return TaskRecapList{}, ErrNotFound
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Timestamp < matched[j].Timestamp
	})

	titles := l.factTitlesFromConfig(ctx, scope, addr)
	items := make([]TaskRecap, 0, len(matched))
	for _, sub := range matched {
		userName := strings.TrimSpace(sub.UserName)
		if userName == "" {
			userName = strings.TrimSpace(sub.UserID)
		}
		recap := TaskRecap{
			JobID:         sub.JobID,
			JobInstanceID: sub.JobInstanceID,
			TaskID:        sub.TaskID,
			Timestamp:     sub.Timestamp,
			UserID:        sub.UserID,
			UserName:      userName,
			Facts:         cloneFacts(sub.Facts),
			Titles:        titles,
		}
		if l.LinkStore != nil {
			links, err := l.LinkStore.Get(ctx, scope, addr, sub.Timestamp)
			if err != nil && !errors.Is(err, ErrNotFound) {
				return TaskRecapList{}, err
			}
			if err == nil {
				recap.Links = links
			}
		}
		items = append(items, recap)
	}
	return TaskRecapList{Items: items}, nil
}

// factTitlesFromConfig reads property titles from the Active Org Config leaf
// schema. Missing config / unknown path → nil (SPA falls back to keys).
func (l *DomainLoader) factTitlesFromConfig(ctx context.Context, scope Scope, addr SubmissionAddress) map[string]string {
	if l.Configs == nil {
		return nil
	}
	version, err := l.Configs.ActiveVersion(ctx, scope)
	if err != nil {
		return nil
	}
	cfg, err := l.Configs.Get(ctx, scope, version)
	if err != nil {
		return nil
	}
	jobVal, ok := cfg.Jobs.Get(addr.JobID)
	if !ok {
		return nil
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil
	}
	leaf, ok := TaskByID(job.Tasks, addr.TaskID)
	if !ok {
		return nil
	}
	payload := taskPayload(leaf)
	if payload == nil {
		return nil
	}
	schema, _ := payload["schema"].(map[string]any)
	return schemaPropertyTitles(schema)
}

func schemaPropertyTitles(schema map[string]any) map[string]string {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]string)
	for key, raw := range props {
		prop, _ := raw.(map[string]any)
		if prop == nil {
			continue
		}
		title, _ := prop["title"].(string)
		title = strings.TrimSpace(title)
		if title != "" {
			out[key] = title
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func submissionsMatching(subs []Submission, addr SubmissionAddress) []Submission {
	addr = addr.normalized()
	var out []Submission
	for _, s := range subs {
		if s.JobID == addr.JobID &&
			s.TaskID == addr.TaskID &&
			s.JobInstanceID == addr.JobInstanceID {
			out = append(out, s)
		}
	}
	return out
}
