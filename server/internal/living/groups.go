package living

import (
	"fmt"
	"strings"
)

// GroupError is a Domain spawn/close failure (not a CommitError).
type GroupError struct {
	Kind    string
	Message string
}

func (e *GroupError) Error() string {
	if e == nil {
		return "living: group error"
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("living: group %s", e.Kind)
}

// SpawnGroupInstance appends an open group instance under a Job instance.
// Rejects duplicate ids and spawn after the Task group is already closed.
func SpawnGroupInstance(domain Domain, jobID, jobInstanceID, groupID, groupInstanceID string) (Domain, error) {
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	groupID = strings.TrimSpace(groupID)
	groupInstanceID = strings.TrimSpace(groupInstanceID)
	if jobID == "" || jobInstanceID == "" || groupID == "" || groupInstanceID == "" {
		return Domain{}, &GroupError{Kind: "invalid", Message: "living: spawn group requires jobId, jobInstanceId, groupId, and groupInstanceId"}
	}
	if IsGroupClosed(domain.ClosedGroups, jobID, jobInstanceID, groupID) {
		return Domain{}, &GroupError{Kind: "already-closed", Message: fmt.Sprintf("living: cannot spawn %s/%s: group %q already closed", jobID, jobInstanceID, groupID)}
	}
	if hasGroupInstance(domain, jobID, jobInstanceID, groupID, groupInstanceID) {
		return Domain{}, &GroupError{Kind: "duplicate-group-instance", Message: fmt.Sprintf("living: group instance %q already exists under %s/%s/%s", groupInstanceID, jobID, jobInstanceID, groupID)}
	}
	next := cloneDomain(domain)
	next.GroupInstances = append(next.GroupInstances, GroupInstance{
		JobID:         jobID,
		JobInstanceID: jobInstanceID,
		GroupID:       groupID,
		ID:            groupInstanceID,
	})
	return next, nil
}

// GroupLifecycleQuery asks spawn/close affordances for one Task group on a Job instance.
type GroupLifecycleQuery struct {
	Group         TaskConfig
	JobID         string
	JobInstanceID string
	Cfg           OrgConfig
	Domain        Domain
}

// GroupLifecycle is canSpawn/canClose for one group on one Job instance.
type GroupLifecycle struct {
	CanSpawn bool
	CanClose bool
}

// EvaluateGroupLifecycle owns canSpawn/canClose once for Worklist chrome —
// no full Worklist projection for two flags.
func EvaluateGroupLifecycle(q GroupLifecycleQuery) GroupLifecycle {
	if q.Group == nil || !q.Group.IsGroup() {
		return GroupLifecycle{}
	}
	groupID := q.Group.ID()
	if groupID == "" {
		return GroupLifecycle{}
	}
	if !groupIsOpen(q, groupID) {
		return GroupLifecycle{}
	}
	n := countGroupInstances(q.Domain.GroupInstances, q.JobID, q.JobInstanceID, groupID)
	repeatable := false
	if v, ok := q.Group["repeatable"].(bool); ok {
		repeatable = v
	}
	return GroupLifecycle{
		CanSpawn: repeatable || n == 0,
		CanClose: groupCloseChildGateMet(q.Group, q.Domain.Submissions, q.JobID, q.JobInstanceID),
	}
}

func groupIsOpen(q GroupLifecycleQuery, groupID string) bool {
	ev := EvaluatePathUnlock(q.Cfg, q.Domain, TaskPathRef{JobID: q.JobID, TaskID: groupID}, q.JobInstanceID)
	return ev.Unlocked && !ev.Done
}

func countGroupInstances(gis []GroupInstance, jobID, jobInstanceID, groupID string) int {
	n := 0
	for _, gi := range gis {
		if gi.JobID == jobID && gi.JobInstanceID == jobInstanceID && gi.GroupID == groupID {
			n++
		}
	}
	return n
}

// groupCloseChildGateMet is the Domain close child-submission gate shared with canClose.
func groupCloseChildGateMet(group TaskConfig, subs []Submission, jobID, jobInstanceID string) bool {
	if !closeRequiresChildSubmission(group) {
		return true
	}
	return hasAnyGroupChildSubmission(subs, jobID, jobInstanceID, group.ID())
}

// CloseGroup closes a Task group on a Job instance. Rejects if already closed.
// When closeRequiresChildSubmission(group), at least one child leaf Submission
// under that group on the Job instance is required.
func CloseGroup(domain Domain, group TaskConfig, jobID, jobInstanceID string) (Domain, error) {
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	if jobID == "" || jobInstanceID == "" {
		return Domain{}, &GroupError{Kind: "invalid", Message: "living: close group requires jobId and jobInstanceId"}
	}
	if group == nil || !group.IsGroup() {
		return Domain{}, &GroupError{Kind: "unknown-group", Message: "living: close requires an authored Task group"}
	}
	groupID := group.ID()
	if groupID == "" {
		return Domain{}, &GroupError{Kind: "unknown-group", Message: "living: group id required"}
	}
	if IsGroupClosed(domain.ClosedGroups, jobID, jobInstanceID, groupID) {
		return Domain{}, &GroupError{Kind: "already-closed", Message: fmt.Sprintf("living: group %q already closed on %s/%s", groupID, jobID, jobInstanceID)}
	}
	if !groupCloseChildGateMet(group, domain.Submissions, jobID, jobInstanceID) {
		return Domain{}, &GroupError{
			Kind:    "missing-child-submission",
			Message: fmt.Sprintf("living: cannot close %q on %s/%s with zero child leaf Submissions", groupID, jobID, jobInstanceID),
		}
	}
	next := cloneDomain(domain)
	next.ClosedGroups = append(next.ClosedGroups, ClosedGroup{
		JobID:         jobID,
		JobInstanceID: jobInstanceID,
		GroupID:       groupID,
	})
	return next, nil
}

func IsGroupClosed(closed []ClosedGroup, jobID, jobInstanceID, groupID string) bool {
	for _, c := range closed {
		if c.JobID == jobID && c.JobInstanceID == jobInstanceID && c.GroupID == groupID {
			return true
		}
	}
	return false
}

// closeRequiresChildSubmission honors only authored closeRequiresChildSubmission.
// Absent or non-bool Config key means the Close child-submission gate is off.
func closeRequiresChildSubmission(group TaskConfig) bool {
	if group == nil {
		return false
	}
	v, ok := group["closeRequiresChildSubmission"].(bool)
	return ok && v
}

func hasGroupInstance(domain Domain, jobID, jobInstanceID, groupID, id string) bool {
	for _, g := range domain.GroupInstances {
		if g.JobID == jobID && g.JobInstanceID == jobInstanceID && g.GroupID == groupID && g.ID == id {
			return true
		}
	}
	return false
}

func hasAnyGroupChildSubmission(subs []Submission, jobID, jobInstanceID, groupID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.JobInstanceID == jobInstanceID && s.GroupID == groupID && s.GroupInstanceID != "" {
			return true
		}
	}
	return false
}
