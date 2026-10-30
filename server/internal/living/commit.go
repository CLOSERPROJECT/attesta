package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CommitRequest is the Commit wire body (no run id).
type CommitRequest struct {
	TaskPath      string         `json:"taskPath"`
	JobInstanceID string         `json:"jobInstanceId"`
	Value         map[string]any `json:"value"`
}

type CommitResult struct {
	Org           string         `json:"org"`
	JobInstanceID string         `json:"jobInstanceId"`
	Domain        Domain         `json:"domain"`
	Links         []HttpLinkItem `json:"links,omitempty"`
	HttpErrors    []string       `json:"httpErrors,omitempty"`
}

// CommitError maps to HTTP status in the handler.
type CommitError struct {
	Kind    string
	Message string
}

func (e *CommitError) Error() string {
	if e.Message == "" {
		return "living: " + e.Kind
	}
	return "living: " + e.Kind + ": " + e.Message
}

type Committer struct {
	Domains    DomainStore
	Configs    ConfigStore
	Authorizer Authorizer
	// HttpPort runs onSubmit.http after DomainStore.Put. Nil is always a no-op
	// (no links/errors) even when the Task declares effects; product wiring plants FakeHttpPort.
	HttpPort HttpPort
	Now      func() time.Time
}

func (c *Committer) Commit(ctx context.Context, org string, op OperatorIdentity, req CommitRequest) (CommitResult, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return CommitResult{}, &CommitError{Kind: "forbidden", Message: "organization required"}
	}
	if c.Domains == nil || c.Configs == nil || c.Authorizer == nil {
		return CommitResult{}, fmt.Errorf("living: commit stores/authorizer required")
	}

	taskPath := strings.TrimSpace(req.TaskPath)
	ref, err := ParseTaskPath(taskPath)
	if err != nil {
		return CommitResult{}, &CommitError{Kind: "unknown-target", Message: err.Error()}
	}
	jobInstanceID := strings.TrimSpace(req.JobInstanceID)
	if jobInstanceID == "" {
		return CommitResult{}, &CommitError{Kind: "prep", Message: "jobInstanceId required"}
	}
	if req.Value == nil {
		return CommitResult{}, &CommitError{Kind: "prep", Message: "value required"}
	}

	version, err := c.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return CommitResult{}, err
	}
	cfg, err := c.Configs.Get(ctx, org, version)
	if err != nil {
		return CommitResult{}, err
	}

	job, task, err := lookupTask(cfg, ref.JobID, ref.TaskID)
	if err != nil {
		return CommitResult{}, &CommitError{Kind: "unknown-target", Message: err.Error()}
	}
	transform, hasTransform := taskOnSubmitTransform(task)
	if !hasTransform {
		return CommitResult{}, &CommitError{
			Kind:    "unknown-target",
			Message: fmt.Sprintf("Task %q has no onSubmit.transform", taskPath),
		}
	}

	domain, err := c.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return CommitResult{}, err
		}
		domain = EmptyDomain()
	}

	allowed, err := c.Authorizer.AllowCommit(ctx, AuthorizeCommit{
		Operator:      op,
		Org:           org,
		TaskPath:      taskPath,
		JobID:         ref.JobID,
		TaskID:        ref.TaskID,
		JobInstanceID: jobInstanceID,
		RolesAllowed:  rolesAllowedForTask(cfg, ref.JobID, ref.TaskID),
		SequenceOk:    sequenceOkForCommit(cfg, domain, ref.JobID, ref.TaskID, jobInstanceID),
		ReadyWhenOk:   readyWhenOkForCommit(domain, task, jobInstanceID),
	})
	if err != nil {
		return CommitResult{}, err
	}
	if !allowed {
		return CommitResult{}, &CommitError{Kind: "unauthorized"}
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}
	ts := now().UTC().Format(time.RFC3339Nano)

	applied, err := applyConfigCommit(domain, cfg, job, task, ref.JobID, ref.TaskID, jobInstanceID, op.UserID, req.Value, transform, ts)
	if err != nil {
		var ce *CommitError
		if errors.As(err, &ce) {
			return CommitResult{}, ce
		}
		var te TransformError
		if errors.As(err, &te) {
			return CommitResult{}, &CommitError{Kind: "prep", Message: te.Error()}
		}
		var ie InstanceExprError
		if errors.As(err, &ie) {
			return CommitResult{}, &CommitError{Kind: "prep", Message: ie.Error()}
		}
		var me MutateError
		if errors.As(err, &me) {
			return CommitResult{}, &CommitError{Kind: "mutate", Message: me.Error()}
		}
		return CommitResult{}, err
	}

	if err := c.Domains.Put(ctx, org, applied.Domain); err != nil {
		return CommitResult{}, err
	}

	result := CommitResult{Org: org, JobInstanceID: applied.JobInstanceID, Domain: applied.Domain}
	if c.HttpPort != nil {
		if effects := taskOnSubmitHTTP(task); len(effects) > 0 {
			httpResult := runSubmitHTTPEffects(ctx, c.HttpPort, effects, applied.Facts, applied.Bindings, applied.Domain, applied.JobInstanceID)
			result.Links = httpResult.Links
			result.HttpErrors = httpResult.Errors
		}
	}
	return result, nil
}

type appliedCommit struct {
	Domain        Domain
	JobInstanceID string
	Facts         map[string]any
	Bindings      ExprBindings
}

func applyConfigCommit(
	domain Domain,
	cfg OrgConfig,
	job Job,
	task map[string]any,
	jobID, taskID, jobInstanceID, userID string,
	value map[string]any,
	transform any,
	ts string,
) (appliedCommit, error) {
	var leftoverIDs []string
	if needsLeftoverProjection(task) {
		ids, err := leftoverIDsForTask(domain, jobID, task, jobInstanceID)
		if err != nil {
			return appliedCommit{}, &CommitError{Kind: "prep", Message: err.Error()}
		}
		leftoverIDs = ids
	}

	prep, err := prepTaskSubmission(task, value, leftoverIDs)
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
	})
	if err != nil {
		return appliedCommit{}, err
	}

	req := run.Request
	req.Timestamp = ts
	req.UserID = userID

	resultInstanceID := jobInstanceID
	if shouldMintJobInstance(job, jobID, taskID, jobInstanceID, domain) {
		id, label, err := EvaluateJobInstanceExprs(job, prep.Facts, run.Bindings)
		if err != nil {
			return appliedCommit{}, err
		}
		// Config instanceId/label are authority (not client-provisional jobInstanceId).
		req.NewJobInstance = &JobInstance{JobID: jobID, ID: id, Label: label}
		req.JobInstanceID = id
		resultInstanceID = id
	}

	next, err := Mutate(domain, req)
	if err != nil {
		return appliedCommit{}, err
	}
	return appliedCommit{
		Domain:        next,
		JobInstanceID: resultInstanceID,
		Facts:         prep.Facts,
		Bindings:      run.Bindings,
	}, nil
}

func intFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case float32:
		if n != float32(int(n)) {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}
