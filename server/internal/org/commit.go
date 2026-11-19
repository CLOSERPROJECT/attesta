package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MintKind selects Job-instance mint behavior on Commit.
type MintKind string

const (
	MintKindNone        MintKind = "none"
	MintKindStartingNew MintKind = "startingNew"
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
	// StreamInstanceID is the Org stream instance scope from the HTTP API (not JSON).
	StreamInstanceID string `json:"-"`
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
// from JSON. Task mounts use addressed GET /api/org/task-form.
type CommitResult struct {
	Org           string   `json:"org"`
	JobInstanceID string   `json:"jobInstanceId"`
	Domain        Domain   `json:"-"`
	Worklist      Worklist `json:"worklist"`
	// Navigation is always set: task, home, or stay (legacy overview → home).
	Navigation CommitNext `json:"navigation"`
	// ContinueNavigation is set only when Navigation is stay (deferred Continue).
	// Never overview — remaining-work / nested defaults map to home.
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
		return "org: " + e.Kind
	}
	return "org: " + e.Kind + ": " + e.Message
}

type TaskCommitter struct {
	Domains    DomainStore
	Configs    ConfigStore
	Authorizer Authorizer
	// HttpPort runs onSubmit.http after the Domain write. Nil is always a no-op
	// (no links/errors) even when the Task declares effects; product uses SidecarHttpPort or soft-fail.
	HttpPort HttpPort
	// SubmitBindings are ops/runtime JSONata vars merged into onSubmit.http evaluation.
	// Stream-scoped $qrEndpoint is always merged from OrgAppBaseURL(scope.Stream) and
	// wins over any qrEndpoint planted here. Nil/empty is fine.
	SubmitBindings ExprBindings
	// LinkStore persists Commit links[] after HttpPort. Nil skips persist; Put errors
	// never fail Commit or roll back Domain (response still returns links for stay UI).
	LinkStore LinkStore
	// DPPs persists Passport records when onSubmit.passport is set. Required for that hook.
	DPPs DPPStore
	Now  func() time.Time
}

func (c *TaskCommitter) Commit(ctx context.Context, scope Scope, op OperatorIdentity, req CommitRequest) (CommitResult, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return CommitResult{}, &CommitError{Kind: "forbidden", Message: "organization and stream required"}
	}
	org := scope.Org
	if c.Domains == nil || c.Configs == nil || c.Authorizer == nil {
		return CommitResult{}, fmt.Errorf("org: commit stores/authorizer required")
	}

	taskPath := strings.TrimSpace(req.TaskPath)
	ref, err := ParseTaskPath(taskPath)
	if err != nil {
		return CommitResult{}, &CommitError{Kind: "unknown-target", Message: err.Error()}
	}
	mintKind := req.mintKind()
	switch mintKind {
	case MintKindNone, MintKindStartingNew:
	default:
		return CommitResult{}, &CommitError{Kind: "prep", Message: "mint.kind must be startingNew or none"}
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
		// Client id is ignored; authorize with empty instance (first Task only).
		jobInstanceID = ""
	case MintKindNone:
		if jobInstanceID == "" {
			return CommitResult{}, &CommitError{Kind: "prep", Message: "jobInstanceId required"}
		}
	}
	if req.Value == nil {
		return CommitResult{}, &CommitError{Kind: "prep", Message: "value required"}
	}

	version, err := c.Configs.ActiveVersion(ctx, scope)
	if err != nil {
		return CommitResult{}, err
	}
	cfg, err := c.Configs.Get(ctx, scope, version)
	if err != nil {
		return CommitResult{}, err
	}
	if err := ValidateStreamConfig(cfg); err != nil {
		return CommitResult{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	if err := ValidateDPPConfig(cfg); err != nil {
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
	if req.StreamInstanceID != "" && jobInstanceID != "" {
		domainForMember, err := c.Domains.Get(ctx, scope)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return CommitResult{}, err
			}
			domainForMember = EmptyDomain()
		}
		if !JobInstanceBelongsToStream(cfg, domainForMember, ref.JobID, jobInstanceID, req.StreamInstanceID) {
			return CommitResult{}, &CommitError{
				Kind:    "conflict",
				Message: "job instance is not a member of the stream instance",
			}
		}
	}

	wantsPassport := taskOnSubmitDPP(task)

	now := c.Now
	if now == nil {
		now = time.Now
	}
	clock := now().UTC()
	ts := clock.Format(time.RFC3339Nano)
	today := clock.Format("2006-01-02")

	// Gates are re-evaluated on each retry so a concurrent Commit cannot slip past readyWhen.
	var applied appliedCommit
	_, err = c.Domains.Update(ctx, scope, func(domain Domain) (Domain, error) {
		streamID := ResolveStreamInstanceID(cfg, domain, ref.JobID, jobInstanceID)
		if streamID == "" {
			streamID = strings.TrimSpace(req.StreamInstanceID)
		}
		allowed, err := c.Authorizer.AllowCommit(ctx, AuthorizeCommit{
			Operator:      op,
			Org:           org,
			Stream:        scope.Stream,
			TaskPath:      taskPath,
			JobID:         ref.JobID,
			TaskID:        ref.TaskID,
			JobInstanceID: jobInstanceID,
			RolesAllowed:  rolesAllowedForPath(cfg, ref),
			SequenceOk:    sequenceOkForPath(cfg, domain, ref, jobInstanceID),
			ReadyWhenOk: readyWhenOkForCommit(
				domain, task, jobInstanceID, cfg.Catalogs,
				streamBindingFrom(cfg, streamID),
			),
		})
		if err != nil {
			return Domain{}, err
		}
		if !allowed {
			return Domain{}, &CommitError{Kind: "unauthorized"}
		}
		applied, err = applyConfigCommit(domain, cfg, job, task, ref, jobInstanceID, op.UserID, req.Value, transform, ts, today, mintKind)
		if err != nil {
			return Domain{}, err
		}
		if wantsPassport {
			if c.DPPs == nil {
				return Domain{}, &CommitError{Kind: "prep", Message: "passport store not configured"}
			}
			// Fail closed before the Domain write so a second lock does not rewrite Domain then error.
			if _, err := c.DPPs.Get(ctx, scope, applied.JobInstanceID); err == nil {
				return Domain{}, &CommitError{Kind: "passport-exists", Message: "passport already issued for subject"}
			} else if !errors.Is(err, ErrNotFound) {
				return Domain{}, err
			}
		}
		return applied.Domain, nil
	})
	if errors.Is(err, ErrDomainConflict) {
		return CommitResult{}, &CommitError{Kind: "conflict", Message: "the data changed while saving; please retry"}
	}
	if err != nil {
		return CommitResult{}, err
	}

	streamInstanceID := strings.TrimSpace(req.StreamInstanceID)
	if streamInstanceID == "" && mintKind == MintKindStartingNew && cfg.Stream != nil && ref.JobID == cfg.Stream.Instance.Job {
		streamInstanceID = applied.JobInstanceID
	}
	var wl Worklist
	if streamInstanceID == "" {
		wl = projectWorklistUnfiltered(cfg, applied.Domain, op)
	} else {
		wl = ProjectWorklist(cfg, applied.Domain, op, streamInstanceID)
	}
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
		issued, err := issueDPP(ctx, c.DPPs, cfg, applied.Domain, applied.JobInstanceID, cfg.Catalogs, applied.Bindings, ts)
		if err != nil {
			return CommitResult{}, err
		}
		result.Links = append(result.Links, issued.Links...)
	}
	if c.HttpPort != nil {
		if effects := taskOnSubmitHTTP(task); len(effects) > 0 {
			httpBindings := mergeBindings(applied.Bindings, c.SubmitBindings)
			httpBindings = mergeBindings(httpBindings, SubmitBindingsForStream(scope.Stream))
			httpResult := runSubmitHTTPEffects(ctx, c.HttpPort, effects, applied.Facts, httpBindings, applied.Domain, applied.JobInstanceID, cfg.Catalogs, org)
			result.Links = append(result.Links, httpResult.Links...)
			result.HttpErrors = httpResult.Errors
		}
	}
	if c.LinkStore != nil && len(result.Links) > 0 {
		// Side record only — never roll back Domain or fail Commit on store error.
		_ = c.LinkStore.Put(ctx, scope, applied.submissionAddress(), ts, result.Links)
	}
	return result, nil
}

// resolveCommitDestination evaluates authored onCommit.navigation first, then
// the default nested/advance/home law (ADR-0006 + advanceTarget). Authored
// overview maps to home on the wire (route kept, unlinked).
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
			if leaf, ok := TaskByID(job.Tasks, ref.TaskID); ok {
				authored = taskCommitNavigation(map[string]any(leaf))
			}
		}
	}
	defaultNav, defaultBundle, err := resolveDefaultCommitDestination(cfg, domain, wl, ref, jobInstanceID)
	if err != nil {
		return "", "", nil, err
	}
	switch authored {
	case commitNavHome, commitNavOverview:
		return CommitNextHome, "", nil, nil
	case commitNavStay:
		return CommitNextStay, defaultNav, defaultBundle, nil
	default:
		return defaultNav, "", defaultBundle, nil
	}
}

// resolveDefaultCommitDestination: auto-advance (skipping !advanceTarget) →
// home when remaining work or finished (ADR-0006 hard cut: overview is not a
// Commit destination).
func resolveDefaultCommitDestination(
	cfg OrgConfig,
	domain Domain,
	wl Worklist,
	ref TaskPathRef,
	jobInstanceID string,
) (CommitNext, *TaskFormBundle, error) {
	jobID := ref.JobID
	if advance, ok := NextAdvanceTaskRef(cfg, wl, jobID, jobInstanceID); ok {
		b, err := BuildTaskFormBundle(cfg, domain, advance)
		if err != nil {
			return "", nil, err
		}
		return CommitNextTask, b, nil
	}
	return CommitNextHome, nil, nil
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
	leaf, ok := TaskByID(job.Tasks, ref.TaskID)
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
