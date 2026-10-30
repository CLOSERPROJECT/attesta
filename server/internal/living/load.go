package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadResult is the SPA hydrate for an org: Worklist + Task form bundles +
// Active Config version metadata. Full Domain / Org Config stay server-side
// (json:"-" for Go callers); they are not on the wire.
type LoadResult struct {
	Org                 string            `json:"org"`
	ActiveConfigVersion string            `json:"activeConfigVersion"`
	ConfigID            string            `json:"configId"`
	Domain              Domain            `json:"-"`
	Config              OrgConfig         `json:"-"`
	Worklist            Worklist          `json:"worklist"`
	TaskFormBundles     []*TaskFormBundle `json:"taskFormBundles"`
}

// Loader hydrates Domain + Active Org Config for an org.
type Loader struct {
	Domains DomainStore
	Configs ConfigStore
}

// Load returns the org Domain (empty when none stored) and the Active Config version document.
func (l *Loader) Load(ctx context.Context, org string) (LoadResult, error) {
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

	wl := ProjectWorklist(cfg, domain)
	bundles, err := ProjectAvailableBundles(cfg, domain, wl)
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
