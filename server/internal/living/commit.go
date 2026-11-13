package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MintKind selects Job-instance or group-instance mint behavior on Commit.
type MintKind string

const (
	MintKindNone        MintKind = "none"
	MintKindStartingNew MintKind = "startingNew"
	MintKindSpawningNew MintKind = "spawningNew"
)

// CommitMint is the Commit wire mint knob (omit → none).
type CommitMint struct {
	Kind MintKind `json:"kind"`
}

// CommitRequest is the Commit wire body (no run id).
type CommitRequest struct {
	TaskPath      string         `json:"taskPath"`
	JobInstanceID string         `json:"jobInstanceId"`
	Value         map[string]any `json:"value"`
	Mint          *CommitMint    `json:"mint,omitempty"`
}

func (req CommitRequest) mintKind() MintKind {
	if req.Mint == nil || req.Mint.Kind == "" {
		return MintKindNone
	}
	return req.Mint.Kind
}

func startingNewMint() *CommitMint {
	return &CommitMint{Kind: MintKindStartingNew}
}

func spawningNewMint() *CommitMint {
	return &CommitMint{Kind: MintKindSpawningNew}
}

// CommitNext is the ADR-0006 post-Commit destination (SPA navigates; does not re-derive).
type CommitNext string

const (
	CommitNextTask     CommitNext = "task"
	CommitNextOverview CommitNext = "overview"
	CommitNextHome     CommitNext = "home"
	CommitNextStay     CommitNext = "stay"
)

// CommitResult is the SPA Commit response: viewer-scoped Worklist, explicit
// navigation destination, and optional Task form bundle when advancing (or
// Continue targeting) a Task. Domain stays on the Go value for tests; omitted
// from JSON. Task mounts use addressed GET /api/living/task-form.
type CommitResult struct {
	Org           string   `json:"org"`
	JobInstanceID string   `json:"jobInstanceId"`
	Domain        Domain   `json:"-"`
	Worklist      Worklist `json:"worklist"`
	// Navigation is always set: task, overview, home, or stay.
	Navigation CommitNext `json:"navigation"`
	// ContinueNavigation is set only when Navigation is stay (deferred Continue).
	ContinueNavigation CommitNext `json:"continueNavigation,omitempty"`
	// NextTaskFormBundle when Navigation or ContinueNavigation is task.
	NextTaskFormBundle *TaskFormBundle `json:"nextTaskFormBundle,omitempty"`
	Links              []HttpLinkItem  `json:"links,omitempty"`
	HttpErrors         []string        `json:"httpErrors,omitempty"`
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
	// (no links/errors) even when the Task declares effects; product uses SidecarHttpPort or soft-fail.
	HttpPort HttpPort
	// SubmitBindings are ops/runtime JSONata vars merged into onSubmit.http evaluation
	// (e.g. qrEndpoint from LivingAppBaseURL). Nil/empty is fine.
	SubmitBindings ExprBindings
	// LinkStore persists Commit links[] after HttpPort. Nil skips persist; Put errors
	// never fail Commit or roll back Domain (response still returns links for stay UI).
	LinkStore LinkStore
	// Passports persists Passport records when onSubmit.passport is set. Required for that hook.
	Passports PassportStore
	Now       func() time.Time
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
	mintKind := req.mintKind()
	switch mintKind {
	case MintKindNone, MintKindStartingNew, MintKindSpawningNew:
	default:
		return CommitResult{}, &CommitError{Kind: "prep", Message: "mint.kind must be startingNew, spawningNew, or none"}
	}
	jobInstanceID := strings.TrimSpace(req.JobInstanceID)
	if jobInstanceID == "_new" {
		return CommitResult{}, &CommitError{
			Kind:    "prep",
			Message: "jobInstanceId _new is not allowed; use mint.startingNew with empty jobInstanceId",
		}
	}
	switch mintKind {
	case MintKindStartingNew:
		if ref.Nested() {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "mint.startingNew is for Job instances; use mint.spawningNew for groups"}
		}
		// Client id is ignored; authorize/Unlock with empty instance (first Task only).
		jobInstanceID = ""
	case MintKindSpawningNew:
		if !ref.Nested() {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "mint.spawningNew requires a nested task path"}
		}
		if jobInstanceID == "" {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "jobInstanceId required to spawn a group instance"}
		}
		if ref.GroupInstanceID != ProvisionalGroupInstanceID {
			return CommitResult{}, &CommitError{
				Kind:    "prep",
				Message: "mint.spawningNew requires group instance id " + ProvisionalGroupInstanceID + " in the task path",
			}
		}
	case MintKindNone:
		if jobInstanceID == "" {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "jobInstanceId required"}
		}
		if ref.Nested() && (ref.GroupInstanceID == "" || ref.GroupInstanceID == ProvisionalGroupInstanceID) {
			return CommitResult{}, &CommitError{
				Kind:    "prep",
				Message: "nested commit requires a real groupInstanceId; use mint.spawningNew with " + ProvisionalGroupInstanceID,
			}
		}
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
	if err := ValidatePassportConfig(cfg); err != nil {
		return CommitResult{}, &CommitError{Kind: "prep", Message: err.Error()}
	}

	job, task, err := lookupLeafTask(cfg, ref)
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
	wantsPassport := taskOnSubmitPassport(task)

	domain, err := c.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return CommitResult{}, err
		}
		domain = EmptyDomain()
	}

	authRef := ref
	if mintKind == MintKindSpawningNew {
		authRef.GroupInstanceID = ProvisionalGroupInstanceID
	}
	allowed, err := c.Authorizer.AllowCommit(ctx, AuthorizeCommit{
		Operator:      op,
		Org:           org,
		TaskPath:      taskPath,
		JobID:         ref.JobID,
		TaskID:        ref.TaskID,
		JobInstanceID: jobInstanceID,
		RolesAllowed:  rolesAllowedForPath(cfg, ref),
		SequenceOk:    sequenceOkForPath(cfg, domain, authRef, jobInstanceID),
		ReadyWhenOk:   readyWhenOkForCommit(domain, task, jobInstanceID, cfg.Catalogs),
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

	applied, err := applyConfigCommit(domain, cfg, job, task, ref, jobInstanceID, op.UserID, req.Value, transform, ts, mintKind)
	if err != nil {
		var ce *CommitError
		if errors.As(err, &ce) {
			return CommitResult{}, ce
		}
		var ge *GroupError
		if errors.As(err, &ge) {
			return CommitResult{}, &CommitError{Kind: "prep", Message: ge.Error()}
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

	if wantsPassport {
		if c.Passports == nil {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "passport store not configured"}
		}
		// Fail closed before Domain Put so a second lock does not rewrite Domain then error.
		if _, err := c.Passports.Get(ctx, org, applied.JobInstanceID); err == nil {
			return CommitResult{}, &CommitError{Kind: "passport-exists", Message: "passport already issued for subject"}
		} else if !errors.Is(err, ErrNotFound) {
			return CommitResult{}, err
		}
	}

	if err := c.Domains.Put(ctx, org, applied.Domain); err != nil {
		return CommitResult{}, err
	}

	wl := ProjectWorklist(cfg, applied.Domain, op)
	nav, continueNav, nextBundle, err := resolveCommitDestination(cfg, applied.Domain, wl, ref, applied.JobInstanceID)
	if err != nil {
		return CommitResult{}, err
	}
	result := CommitResult{
		Org:                org,
		JobInstanceID:      applied.JobInstanceID,
		Domain:             applied.Domain,
		Worklist:           wl,
		Navigation:         nav,
		ContinueNavigation: continueNav,
		NextTaskFormBundle: nextBundle,
	}
	if wantsPassport {
		issued, err := issuePassport(ctx, c.Passports, cfg, applied.Domain, applied.JobInstanceID, cfg.Catalogs, applied.Bindings, ts)
		if err != nil {
			return CommitResult{}, err
		}
		result.Links = append(result.Links, issued.Links...)
	}
	if c.HttpPort != nil {
		if effects := taskOnSubmitHTTP(task); len(effects) > 0 {
			httpBindings := mergeBindings(applied.Bindings, c.SubmitBindings)
			httpResult := runSubmitHTTPEffects(ctx, c.HttpPort, effects, applied.Facts, httpBindings, applied.Domain, applied.JobInstanceID, cfg.Catalogs, org)
			result.Links = append(result.Links, httpResult.Links...)
			result.HttpErrors = httpResult.Errors
		}
	}
	if c.LinkStore != nil && len(result.Links) > 0 {
		// Side record only — never roll back Domain or fail Commit on store error.
		_ = c.LinkStore.Put(ctx, org, applied.submissionAddress(), result.Links)
	}
	return result, nil
}

// resolveCommitDestination evaluates authored onCommit.navigation first, then
// the default nested/advance/overview/home law (ADR-0006 + advanceTarget).
func resolveCommitDestination(
	cfg OrgConfig,
	domain Domain,
	wl Worklist,
	ref TaskPathRef,
	jobInstanceID string,
) (nav CommitNext, continueNav CommitNext, bundle *TaskFormBundle, err error) {
	authored := commitNavAdvance
	if jobVal, ok := cfg.Jobs.Get(ref.JobID); ok {
		if job, ok := AsJob(jobVal); ok {
			if leaf, ok := LeafTaskByPath(job.Tasks, ref); ok {
				authored = taskCommitNavigation(map[string]any(leaf))
			}
		}
	}
	defaultNav, defaultBundle, err := resolveDefaultCommitDestination(cfg, domain, wl, ref, jobInstanceID)
	if err != nil {
		return "", "", nil, err
	}
	switch authored {
	case commitNavHome:
		return CommitNextHome, "", nil, nil
	case commitNavOverview:
		return CommitNextOverview, "", nil, nil
	case commitNavStay:
		return CommitNextStay, defaultNav, defaultBundle, nil
	default:
		return defaultNav, "", defaultBundle, nil
	}
}

// resolveDefaultCommitDestination: nested child while group open → overview;
// else auto-advance (skipping !advanceTarget) → overview → home.
func resolveDefaultCommitDestination(
	cfg OrgConfig,
	domain Domain,
	wl Worklist,
	ref TaskPathRef,
	jobInstanceID string,
) (CommitNext, *TaskFormBundle, error) {
	jobID := ref.JobID
	if ref.Nested() && !IsGroupClosed(domain.ClosedGroups, jobID, jobInstanceID, ref.GroupID) {
		return CommitNextOverview, nil, nil
	}
	if advance, ok := NextAdvanceTaskRef(cfg, wl, jobID, jobInstanceID); ok {
		b, err := BuildTaskFormBundle(cfg, domain, advance)
		if err != nil {
			return "", nil, err
		}
		return CommitNextTask, b, nil
	}
	if worklistInstanceHasRemainingWork(wl, jobID, jobInstanceID) {
		return CommitNextOverview, nil, nil
	}
	return CommitNextHome, nil, nil
}

type appliedCommit struct {
	Domain          Domain
	JobInstanceID   string
	GroupID         string
	GroupInstanceID string
	JobID           string
	TaskID          string
	Facts           map[string]any
	Bindings        ExprBindings
}

func (a appliedCommit) submissionAddress() SubmissionAddress {
	return SubmissionAddress{
		JobID:           a.JobID,
		JobInstanceID:   a.JobInstanceID,
		TaskID:          a.TaskID,
		GroupID:         a.GroupID,
		GroupInstanceID: a.GroupInstanceID,
	}
}

func applyConfigCommit(
	domain Domain,
	cfg OrgConfig,
	job Job,
	task map[string]any,
	ref TaskPathRef,
	jobInstanceID, userID string,
	value map[string]any,
	transform any,
	ts string,
	mintKind MintKind,
) (appliedCommit, error) {
	jobID := ref.JobID
	taskID := ref.TaskID
	groupID := ref.GroupID
	groupInstanceID := ref.GroupInstanceID

	var rowIDs []string
	if needsRowProjection(task) {
		ids, err := rowIDsForTask(domain, jobID, task, jobInstanceID, cfg.Catalogs)
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
	})
	if err != nil {
		return appliedCommit{}, err
	}

	req := run.Request
	req.Timestamp = ts
	req.UserID = userID

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
			return appliedCommit{}, err
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

	working := domain
	if mintKind == MintKindSpawningNew {
		group, ok := TaskByID(job.Tasks, groupID)
		if !ok || !group.IsGroup() {
			return appliedCommit{}, &CommitError{Kind: "unknown-target", Message: fmt.Sprintf("unknown Task group %q", groupID)}
		}
		ids := OrderedTaskIDs(group.ChildTasks())
		if len(ids) == 0 || ids[0] != taskID {
			return appliedCommit{}, &CommitError{
				Kind:    "prep",
				Message: "mint.spawningNew requires the first leaf in the Task group",
			}
		}
		groupInstanceID = nextGroupInstanceID(working, jobID, jobInstanceID, groupID)
		spawned, err := SpawnGroupInstance(working, jobID, jobInstanceID, groupID, groupInstanceID)
		if err != nil {
			return appliedCommit{}, err
		}
		working = spawned
	}

	if groupID != "" {
		req.GroupID = groupID
		req.GroupInstanceID = groupInstanceID
	}

	next, err := Mutate(working, req)
	if err != nil {
		return appliedCommit{}, err
	}
	return appliedCommit{
		Domain:          next,
		JobInstanceID:   resultInstanceID,
		JobID:           jobID,
		TaskID:          taskID,
		GroupID:         groupID,
		GroupInstanceID: groupInstanceID,
		Facts:           prep.Facts,
		Bindings:        run.Bindings,
	}, nil
}

func nextGroupInstanceID(domain Domain, jobID, jobInstanceID, groupID string) string {
	n := 0
	for _, g := range domain.GroupInstances {
		if g.JobID == jobID && g.JobInstanceID == jobInstanceID && g.GroupID == groupID {
			n++
		}
	}
	return fmt.Sprintf("%s-%d", groupID, n+1)
}

func lookupLeafTask(cfg OrgConfig, ref TaskPathRef) (Job, map[string]any, error) {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return Job{}, nil, fmt.Errorf("unknown Job %q", ref.JobID)
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return Job{}, nil, fmt.Errorf("invalid Job %q", ref.JobID)
	}
	leaf, ok := LeafTaskByPath(job.Tasks, ref)
	if !ok {
		return Job{}, nil, fmt.Errorf("unknown Task %q", ref.String())
	}
	return job, map[string]any(leaf), nil
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
