package org

import (
	"errors"
	"fmt"
)

// appliedCommit is the Domain outcome of applying one Task Commit
// (prep → transform → mint → Mutate): new Domain, bindings, and Job instance id.
type appliedCommit struct {
	Domain        Domain
	JobInstanceID string
	JobID         string
	TaskID        string
	Facts         map[string]any
	Bindings      ExprBindings
}

func (a appliedCommit) submissionAddress() SubmissionAddress {
	return SubmissionAddress{
		JobID:         a.JobID,
		JobInstanceID: a.JobInstanceID,
		TaskID:        a.TaskID,
	}
}

// applyConfigCommit is the apply-Task-Commit-to-Domain interface: sealed value +
// Config/Domain context in → Domain + bindings + Job instance id out. Prep,
// transform op parsing, mint.startingNew, and Mutate stay behind this seam;
// callers see CommitError kinds, not TransformError / InstanceExprError / MutateError.
func applyConfigCommit(
	domain Domain,
	cfg OrgConfig,
	job Job,
	task map[string]any,
	ref TaskPathRef,
	jobInstanceID, userID, userName string,
	value map[string]any,
	transform any,
	ts, today string,
	mintKind MintKind,
) (appliedCommit, error) {
	jobID := ref.JobID
	taskID := ref.TaskID

	stream := streamBindingFrom(cfg, ResolveStreamInstanceID(cfg, domain, jobID, jobInstanceID))
	var rowIDs []string
	if needsRowProjection(task) {
		ids, err := rowIDsForTask(domain, jobID, task, jobInstanceID, cfg.Catalogs, stream)
		if err != nil {
			return appliedCommit{}, &CommitError{Kind: "prep", Message: err.Error()}
		}
		rowIDs = ids
	}

	prep, err := prepTaskSubmission(task, value, rowIDs)
	if err != nil {
		return appliedCommit{}, &CommitError{Kind: "prep", Message: err.Error()}
	}

	run, err := RunTransform(domain, RunTransformInput{
		JobID:         jobID,
		TaskID:        taskID,
		JobInstanceID: jobInstanceID,
		UserID:        userID,
		Facts:         prep.Facts,
		Transform:     transform,
		Allowed:       EntityKindsFromConfig(cfg),
		Catalogs:      cfg.Catalogs,
		Today:         today,
		Stream:        stream,
	})
	if err != nil {
		return appliedCommit{}, mapApplyCommitError(err)
	}

	req := run.Request
	req.Timestamp = ts
	req.UserID = userID
	req.UserName = userName

	resultInstanceID := jobInstanceID
	if mintKind == MintKindStartingNew {
		ids := OrderedTaskIDs(job.Tasks)
		if len(ids) == 0 || ids[0] != taskID {
			return appliedCommit{}, &CommitError{
				Kind:    "prep",
				Message: "mint.startingNew requires the first Task in Job order",
			}
		}
		id, label, err := EvaluateJobInstanceExprs(job, prep.Facts, run.Bindings)
		if err != nil {
			return appliedCommit{}, mapApplyCommitError(err)
		}
		if hasJobInstance(domain, jobID, id) {
			return appliedCommit{}, &CommitError{
				Kind:    "prep",
				Message: fmt.Sprintf("job instance %s/%s already exists", jobID, id),
			}
		}
		req.NewJobInstance = &JobInstance{JobID: jobID, ID: id, Label: label}
		req.JobInstanceID = id
		resultInstanceID = id
	}

	next, err := Mutate(domain, req)
	if err != nil {
		return appliedCommit{}, mapApplyCommitError(err)
	}
	return appliedCommit{
		Domain:        next,
		JobInstanceID: resultInstanceID,
		JobID:         jobID,
		TaskID:        taskID,
		Facts:         prep.Facts,
		Bindings:      run.Bindings,
	}, nil
}

func mapApplyCommitError(err error) error {
	var ce *CommitError
	if errors.As(err, &ce) {
		return ce
	}
	var te TransformError
	if errors.As(err, &te) {
		return &CommitError{Kind: "prep", Message: te.Error()}
	}
	var ie InstanceExprError
	if errors.As(err, &ie) {
		return &CommitError{Kind: "prep", Message: ie.Error()}
	}
	var me MutateError
	if errors.As(err, &me) {
		return &CommitError{Kind: "mutate", Message: me.Error()}
	}
	return err
}
