package living

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
	JobID           string            `json:"jobId"`
	JobInstanceID   string            `json:"jobInstanceId"`
	TaskID          string            `json:"taskId"`
	GroupID         string            `json:"groupId,omitempty"`
	GroupInstanceID string            `json:"groupInstanceId,omitempty"`
	Timestamp       string            `json:"timestamp"`
	UserID          string            `json:"userId"`
	Facts           map[string]any    `json:"facts"`
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
func (l *Loader) LoadTaskRecap(
	ctx context.Context,
	org string,
	addr SubmissionAddress,
) (TaskRecapList, error) {
	org = strings.TrimSpace(org)
	addr = addr.normalized()
	if org == "" {
		return TaskRecapList{}, fmt.Errorf("living: org required")
	}
	if !addr.valid() {
		return TaskRecapList{}, fmt.Errorf("living: submission address required")
	}
	if l.Domains == nil {
		return TaskRecapList{}, fmt.Errorf("living: stores required")
	}

	domain, err := l.Domains.Get(ctx, org)
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

	titles := l.factTitlesFromConfig(ctx, org, addr)
	items := make([]TaskRecap, 0, len(matched))
	for _, sub := range matched {
		recap := TaskRecap{
			JobID:           sub.JobID,
			JobInstanceID:   sub.JobInstanceID,
			TaskID:          sub.TaskID,
			GroupID:         sub.GroupID,
			GroupInstanceID: sub.GroupInstanceID,
			Timestamp:       sub.Timestamp,
			UserID:          sub.UserID,
			Facts:           cloneFacts(sub.Facts),
			Titles:          titles,
		}
		if l.LinkStore != nil {
			links, err := l.LinkStore.Get(ctx, org, addr, sub.Timestamp)
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
func (l *Loader) factTitlesFromConfig(ctx context.Context, org string, addr SubmissionAddress) map[string]string {
	if l.Configs == nil {
		return nil
	}
	version, err := l.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return nil
	}
	cfg, err := l.Configs.Get(ctx, org, version)
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
	leaf, ok := LeafTaskByPath(job.Tasks, TaskPathRef{
		JobID:           addr.JobID,
		GroupID:         addr.GroupID,
		GroupInstanceID: addr.GroupInstanceID,
		TaskID:          addr.TaskID,
	})
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
			s.JobInstanceID == addr.JobInstanceID &&
			s.GroupID == addr.GroupID &&
			s.GroupInstanceID == addr.GroupInstanceID {
			out = append(out, s)
		}
	}
	return out
}
