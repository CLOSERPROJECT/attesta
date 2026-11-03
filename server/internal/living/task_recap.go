package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// TaskRecap is the thin overview expand payload for a done Task: Submission
// facts + actor/time, plus any post-Commit links side record. No schema,
// uiSchema, or Domain.
type TaskRecap struct {
	JobID           string         `json:"jobId"`
	JobInstanceID   string         `json:"jobInstanceId"`
	TaskID          string         `json:"taskId"`
	GroupID         string         `json:"groupId,omitempty"`
	GroupInstanceID string         `json:"groupInstanceId,omitempty"`
	Timestamp       string         `json:"timestamp"`
	UserID          string         `json:"userId"`
	Facts           map[string]any `json:"facts"`
	Links           []HttpLinkItem `json:"links,omitempty"`
}

// LoadTaskRecap returns Submission facts (+ timestamp, userId) and stored links
// for a done Task address. Auth is Caller's concern (Load-level Gate + org);
// this method does not Cerbos-authorize. Unknown address or not-done → ErrNotFound.
func (l *Loader) LoadTaskRecap(
	ctx context.Context,
	org string,
	addr SubmissionAddress,
) (TaskRecap, error) {
	org = strings.TrimSpace(org)
	addr = addr.normalized()
	if org == "" {
		return TaskRecap{}, fmt.Errorf("living: org required")
	}
	if !addr.valid() {
		return TaskRecap{}, fmt.Errorf("living: submission address required")
	}
	if l.Domains == nil {
		return TaskRecap{}, fmt.Errorf("living: stores required")
	}

	domain, err := l.Domains.Get(ctx, org)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return TaskRecap{}, ErrNotFound
		}
		return TaskRecap{}, err
	}

	sub, ok := findSubmission(domain.Submissions, addr)
	if !ok {
		return TaskRecap{}, ErrNotFound
	}

	recap := TaskRecap{
		JobID:           sub.JobID,
		JobInstanceID:   sub.JobInstanceID,
		TaskID:          sub.TaskID,
		GroupID:         sub.GroupID,
		GroupInstanceID: sub.GroupInstanceID,
		Timestamp:       sub.Timestamp,
		UserID:          sub.UserID,
		Facts:           cloneFacts(sub.Facts),
	}

	if l.LinkStore != nil {
		links, err := l.LinkStore.Get(ctx, org, addr)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return TaskRecap{}, err
		}
		if err == nil {
			recap.Links = links
		}
	}
	return recap, nil
}

func findSubmission(subs []Submission, addr SubmissionAddress) (Submission, bool) {
	addr = addr.normalized()
	for _, s := range subs {
		if s.JobID == addr.JobID &&
			s.TaskID == addr.TaskID &&
			s.JobInstanceID == addr.JobInstanceID &&
			s.GroupID == addr.GroupID &&
			s.GroupInstanceID == addr.GroupInstanceID {
			return s, true
		}
	}
	return Submission{}, false
}
