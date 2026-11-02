package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadResult is the SPA hydrate for an org: viewer-scoped Worklist + Task form
// bundles + Active Config version metadata. Full Domain / Org Config stay
// server-side (json:"-" for Go callers); they are not on the wire.
//
// Interim (pre-04 JobPanel): TaskFormBundles still ships every
// CollectAvailableTaskRefs address. ADR-0006 end-state omits bulk once Task
// routes fetch via LoadTaskBundle; the addressed path does not depend on bulk.
type LoadResult struct {
	Org                 string            `json:"org"`
	ActiveConfigVersion string            `json:"activeConfigVersion"`
	ConfigID            string            `json:"configId"`
	Domain              Domain            `json:"-"`
	Config              OrgConfig         `json:"-"`
	Worklist            Worklist          `json:"worklist"`
	TaskFormBundles     []*TaskFormBundle `json:"taskFormBundles"`
}

// ErrTaskBundleForbidden is returned when the viewer cannot role-mount the Task
// (board scope; not Cerbos).
var ErrTaskBundleForbidden = errors.New("living: task form bundle forbidden")

// Loader hydrates Domain + Active Org Config for an org.
type Loader struct {
	Domains DomainStore
	Configs ConfigStore
}

// Load returns the org Domain (empty when none stored) and the Active Config
// version document, projecting Worklist Available / canStart for op's roles.
func (l *Loader) Load(ctx context.Context, org string, op OperatorIdentity) (LoadResult, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return LoadResult{}, fmt.Errorf("living: org required")
	}
	if l.Domains == nil || l.Configs == nil {
		return LoadResult{}, fmt.Errorf("living: stores required")
	}

	version, err := l.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return LoadResult{}, err
	}
	cfg, err := l.Configs.Get(ctx, org, version)
	if err != nil {
		return LoadResult{}, err
	}

	domain, err := l.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return LoadResult{}, err
		}
		domain = EmptyDomain()
	}

	wl := ProjectWorklist(cfg, domain, op)
	// Bulk Available bundles: keep for pre-04 JobPanel; ticket 04 drops once
	// Task routes use LoadTaskBundle.
	bundles, err := ProjectAvailableBundles(cfg, domain, wl, op.Roles)
	if err != nil {
		return LoadResult{}, err
	}
	return LoadResult{
		Org:                 org,
		ActiveConfigVersion: version,
		ConfigID:            cfg.ID,
		Domain:              domain,
		Config:              cfg,
		Worklist:            wl,
		TaskFormBundles:     bundles,
	}, nil
}

// LoadTaskBundle returns one Task form bundle for jobId+taskId+jobInstanceId
// (empty or "new" = provisional start). Denied when outside viewer Worklist scope.
func (l *Loader) LoadTaskBundle(ctx context.Context, org string, op OperatorIdentity, jobID, taskID, jobInstanceID string) (*TaskFormBundle, error) {
	org = strings.TrimSpace(org)
	jobID = strings.TrimSpace(jobID)
	taskID = strings.TrimSpace(taskID)
	jobInstanceID = NormalizeJobInstanceID(jobInstanceID)
	if org == "" || jobID == "" || taskID == "" {
		return nil, fmt.Errorf("living: org, jobId, and taskId required")
	}
	if l.Domains == nil || l.Configs == nil {
		return nil, fmt.Errorf("living: stores required")
	}

	version, err := l.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return nil, err
	}
	cfg, err := l.Configs.Get(ctx, org, version)
	if err != nil {
		return nil, err
	}

	domain, err := l.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		domain = EmptyDomain()
	}

	wl := ProjectWorklist(cfg, domain, op)
	ref := AvailableTaskRef{JobID: jobID, TaskID: taskID, JobInstanceID: jobInstanceID}
	if !ViewerMayMountTask(cfg, wl, ref, op.Roles) {
		return nil, ErrTaskBundleForbidden
	}
	return BuildTaskFormBundle(cfg, domain, ref)
}
