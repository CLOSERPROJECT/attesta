package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadTaskRecap projects the overview table for addr (entity or submission mode)
// and attaches LinkStore links keyed by each row's SubmissionID. Auth is Caller's
// concern (Load-level Gate + org); this method does not Cerbos-authorize.
// Unknown address or no matching Submissions → ErrNotFound.
func (l *DomainLoader) LoadTaskRecap(
	ctx context.Context,
	scope Scope,
	addr SubmissionAddress,
) (TaskRecapTable, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return TaskRecapTable{}, err
	}
	addr = addr.normalized()
	if !addr.valid() {
		return TaskRecapTable{}, fmt.Errorf("org: submission address required")
	}
	if l.Domains == nil || l.Configs == nil {
		return TaskRecapTable{}, fmt.Errorf("org: stores required")
	}

	domain, err := l.Domains.Get(ctx, scope)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return TaskRecapTable{}, ErrNotFound
		}
		return TaskRecapTable{}, err
	}

	matched := submissionsMatching(domain.Submissions, addr)
	if len(matched) == 0 {
		return TaskRecapTable{}, ErrNotFound
	}

	version, err := l.Configs.ActiveVersion(ctx, scope)
	if err != nil {
		return TaskRecapTable{}, err
	}
	cfg, err := l.Configs.Get(ctx, scope, version)
	if err != nil {
		return TaskRecapTable{}, err
	}

	table, err := ProjectTaskRecap(cfg, domain, addr)
	if err != nil {
		return TaskRecapTable{}, err
	}

	if l.LinkStore != nil {
		for i := range table.Rows {
			links, err := l.LinkStore.Get(ctx, scope, addr, table.Rows[i].SubmissionID)
			if err != nil && !errors.Is(err, ErrNotFound) {
				return TaskRecapTable{}, err
			}
			if err == nil {
				table.Rows[i].Links = links
			}
		}
	}
	return table, nil
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
