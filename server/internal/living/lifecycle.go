package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CloseGroupRequest closes a Task group on a Job instance (lifecycle, not Commit).
type CloseGroupRequest struct {
	JobID         string `json:"jobId"`
	JobInstanceID string `json:"jobInstanceId"`
	GroupID       string `json:"groupId"`
}

type groupLifecyclePrep struct {
	Org, JobID, JobInstanceID, GroupID string
	Cfg                                OrgConfig
	Group                              TaskConfig
	Domain                             Domain
	CanClose                           bool
}

// prepGroupLifecycle looks up org/config/job/group for Close.
// Caller enforces CanClose via EvaluateGroupLifecycle
// (Cerbos is leaf-Commit only).
func (c *Committer) prepGroupLifecycle(
	ctx context.Context,
	org string,
	jobID, jobInstanceID, groupID string,
) (groupLifecyclePrep, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return groupLifecyclePrep{}, &CommitError{Kind: "forbidden", Message: "organization required"}
	}
	if c.Domains == nil || c.Configs == nil {
		return groupLifecyclePrep{}, fmt.Errorf("living: close stores required")
	}
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	groupID = strings.TrimSpace(groupID)
	if jobID == "" || jobInstanceID == "" || groupID == "" {
		return groupLifecyclePrep{}, &CommitError{Kind: "prep", Message: "jobId, jobInstanceId, and groupId required"}
	}

	version, err := c.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return groupLifecyclePrep{}, err
	}
	cfg, err := c.Configs.Get(ctx, org, version)
	if err != nil {
		return groupLifecyclePrep{}, err
	}
	jobVal, ok := cfg.Jobs.Get(jobID)
	if !ok {
		return groupLifecyclePrep{}, &CommitError{Kind: "unknown-target", Message: fmt.Sprintf("unknown Job %q", jobID)}
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return groupLifecyclePrep{}, &CommitError{Kind: "unknown-target", Message: fmt.Sprintf("invalid Job %q", jobID)}
	}
	group, ok := TaskByID(job.Tasks, groupID)
	if !ok || !group.IsGroup() {
		return groupLifecyclePrep{}, &CommitError{Kind: "unknown-target", Message: fmt.Sprintf("unknown Task group %q", groupID)}
	}

	domain, err := c.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return groupLifecyclePrep{}, err
		}
		domain = EmptyDomain()
	}
	if !hasJobInstance(domain, jobID, jobInstanceID) {
		return groupLifecyclePrep{}, &CommitError{Kind: "prep", Message: fmt.Sprintf("unknown job instance %s/%s", jobID, jobInstanceID)}
	}

	lc := EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group:         group,
		JobID:         jobID,
		JobInstanceID: jobInstanceID,
		Cfg:           cfg,
		Domain:        domain,
	})
	return groupLifecyclePrep{
		Org:           org,
		JobID:         jobID,
		JobInstanceID: jobInstanceID,
		GroupID:       groupID,
		Cfg:           cfg,
		Group:         group,
		Domain:        domain,
		CanClose:      lc.CanClose,
	}, nil
}

// CloseGroup closes a Task group and returns Commit-shaped next + Worklist.
func (c *Committer) CloseGroup(ctx context.Context, org string, op OperatorIdentity, req CloseGroupRequest) (CommitResult, error) {
	prep, err := c.prepGroupLifecycle(ctx, org, req.JobID, req.JobInstanceID, req.GroupID)
	if err != nil {
		return CommitResult{}, err
	}
	if !prep.CanClose {
		return CommitResult{}, &CommitError{Kind: "prep", Message: fmt.Sprintf("cannot close group %q on %s/%s", prep.GroupID, prep.JobID, prep.JobInstanceID)}
	}

	next, err := CloseGroup(prep.Domain, prep.Group, prep.JobID, prep.JobInstanceID)
	if err != nil {
		var ge *GroupError
		if errors.As(err, &ge) {
			return CommitResult{}, commitErrorFromGroupError(ge)
		}
		return CommitResult{}, err
	}
	if err := c.Domains.Put(ctx, prep.Org, next); err != nil {
		return CommitResult{}, err
	}
	return projectLifecycleResult(prep.Org, prep.JobID, prep.JobInstanceID, prep.Cfg, next, op)
}

func projectLifecycleResult(org, jobID, jobInstanceID string, cfg OrgConfig, domain Domain, op OperatorIdentity) (CommitResult, error) {
	wl := ProjectWorklist(cfg, domain, op)
	// Flat ref: Close is not a nested child Commit; use usual auto-advance law.
	next, nextBundle, err := resolveCommitDestination(cfg, domain, wl, TaskPathRef{JobID: jobID}, jobInstanceID)
	if err != nil {
		return CommitResult{}, err
	}
	return CommitResult{
		Org:                org,
		JobInstanceID:      jobInstanceID,
		Domain:             domain,
		Worklist:           wl,
		Next:               next,
		NextTaskFormBundle: nextBundle,
	}, nil
}

// commitErrorFromGroupError keeps missing-child-submission distinct so clients
// can tell "close blocked: no leaf yet" from generic prep failures.
func commitErrorFromGroupError(ge *GroupError) *CommitError {
	kind := "prep"
	if ge != nil && ge.Kind == "missing-child-submission" {
		kind = "missing-child-submission"
	}
	msg := ""
	if ge != nil {
		msg = ge.Error()
	}
	return &CommitError{Kind: kind, Message: msg}
}
