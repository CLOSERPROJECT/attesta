package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadResult is the SPA hydrate for an org: viewer-scoped Worklist + Active
// Config version metadata (+ OperatorIdentity roles for Available affordances).
// Full Domain / Org Config stay server-side (json:"-" for Go callers); they are
// not on the wire. ADR-0006: home Load omits bulk Task form bundles — Task
// routes fetch via LoadTaskBundle; Commit returns next + nextTaskFormBundle when advancing.
type LoadResult struct {
	Org                 string `json:"org"`
	ActiveConfigVersion string `json:"activeConfigVersion"`
	ConfigID            string `json:"configId"`
	// Viewer OperatorIdentity projection for SPA chrome ($operator) — not Cerbos.
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	// Roles are the caller's Config role slugs (OperatorIdentity; not Cerbos).
	Roles []string `json:"roles"`
	// Revision is the Domain revision this Load reflects (lower bound).
	Revision int64     `json:"revision"`
	Domain   Domain    `json:"-"`
	Config   OrgConfig `json:"-"`
	Worklist Worklist  `json:"worklist"`
	// Empty on home Load (ADR-0006). Kept on the wire as [] for stable JSON shape.
	TaskFormBundles []*TaskFormBundle `json:"taskFormBundles"`
}

// ErrTaskBundleForbidden is returned when the viewer cannot role-mount the Task
// (board scope; not Cerbos).
var ErrTaskBundleForbidden = errors.New("org: task form bundle forbidden")

// DomainLoader hydrates Domain + Active Org Config for an org.
type DomainLoader struct {
	Domains DomainStore
	Configs ConfigStore
	// LinkStore supplies post-Commit links for TaskRecapTable rows. Nil → cells only.
	LinkStore LinkStore
}

// Load returns the stream Domain (empty when none stored) and the Active Config
// version document, projecting the unfiltered Worklist. Prefer LoadForStreamInstance
// when an Org stream instance id is known (§5).
func (l *DomainLoader) Load(ctx context.Context, scope Scope, op OperatorIdentity) (LoadResult, error) {
	return l.LoadForStreamInstance(ctx, scope, op, "")
}

// LoadForStreamInstance is Load with Worklist filtered to streamInstanceID when non-empty.
func (l *DomainLoader) LoadForStreamInstance(ctx context.Context, scope Scope, op OperatorIdentity, streamInstanceID string) (LoadResult, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return LoadResult{}, err
	}
	if l.Domains == nil || l.Configs == nil {
		return LoadResult{}, fmt.Errorf("org: stores required")
	}

	version, err := l.Configs.ActiveVersion(ctx, scope)
	if err != nil {
		return LoadResult{}, err
	}
	cfg, err := l.Configs.Get(ctx, scope, version)
	if err != nil {
		return LoadResult{}, err
	}

	domain, revision, err := domainSnapshot(ctx, l.Domains, scope)
	if err != nil {
		return LoadResult{}, err
	}

	var wl Worklist
	if strings.TrimSpace(streamInstanceID) == "" {
		wl = projectWorklistUnfiltered(cfg, domain, op)
	} else {
		wl = ProjectWorklist(cfg, domain, op, streamInstanceID)
	}
	displayName := strings.TrimSpace(op.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(op.UserID)
	}
	return LoadResult{
		Org:                 scope.Org,
		ActiveConfigVersion: version,
		ConfigID:            cfg.ID,
		UserID:              strings.TrimSpace(op.UserID),
		DisplayName:         displayName,
		Roles:               append([]string(nil), op.Roles...),
		Revision:            revision,
		Domain:              domain,
		Config:              cfg,
		Worklist:            wl,
		TaskFormBundles:     []*TaskFormBundle{},
	}, nil
}

// domainSnapshot reads the revision before the Domain, so the reported revision
// is never newer than the data returned. A stale report costs the client one extra
// refresh; the opposite order could hide a write.
func domainSnapshot(ctx context.Context, domains DomainStore, scope Scope) (Domain, int64, error) {
	revision, err := domains.Revision(ctx, scope)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Domain{}, 0, err
	}
	domain, err := domains.Get(ctx, scope)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return Domain{}, 0, err
		}
		domain = EmptyDomain()
	}
	return domain, revision, nil
}

// LoadTaskBundle returns one Task form bundle for jobId+taskId+jobInstanceId
// (empty or "new" = provisional start). Denied when outside viewer Worklist scope.
func (l *DomainLoader) LoadTaskBundle(
	ctx context.Context,
	scope Scope,
	op OperatorIdentity,
	jobID, taskID, jobInstanceID string,
) (*TaskFormBundle, error) {
	return l.LoadTaskBundleForStreamInstance(ctx, scope, op, jobID, taskID, jobInstanceID, "")
}

// LoadTaskBundleForStreamInstance is LoadTaskBundle with Worklist filtered to streamInstanceID when set.
func (l *DomainLoader) LoadTaskBundleForStreamInstance(
	ctx context.Context,
	scope Scope,
	op OperatorIdentity,
	jobID, taskID, jobInstanceID, streamInstanceID string,
) (*TaskFormBundle, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return nil, err
	}
	jobID = strings.TrimSpace(jobID)
	taskID = strings.TrimSpace(taskID)
	jobInstanceID = NormalizeJobInstanceID(jobInstanceID)
	if jobID == "" || taskID == "" {
		return nil, fmt.Errorf("org: org, jobId, and taskId required")
	}
	if l.Domains == nil || l.Configs == nil {
		return nil, fmt.Errorf("org: stores required")
	}

	version, err := l.Configs.ActiveVersion(ctx, scope)
	if err != nil {
		return nil, err
	}
	cfg, err := l.Configs.Get(ctx, scope, version)
	if err != nil {
		return nil, err
	}

	domain, err := l.Domains.Get(ctx, scope)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		domain = EmptyDomain()
	}

	var wl Worklist
	if strings.TrimSpace(streamInstanceID) == "" {
		wl = projectWorklistUnfiltered(cfg, domain, op)
	} else {
		wl = ProjectWorklist(cfg, domain, op, streamInstanceID)
	}
	ref := ActionableTaskRef{
		JobID:            jobID,
		TaskID:           taskID,
		JobInstanceID:    jobInstanceID,
		StreamInstanceID: strings.TrimSpace(streamInstanceID),
	}
	if !ViewerMayMountTask(cfg, wl, ref) {
		return nil, ErrTaskBundleForbidden
	}
	return BuildTaskFormBundle(cfg, domain, ref)
}
