package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadResult is the Domain + Active Config version hydrate for an org.
type LoadResult struct {
	Org                 string    `json:"org"`
	ActiveConfigVersion string    `json:"activeConfigVersion"`
	Domain              Domain    `json:"domain"`
	Config              OrgConfig `json:"config"`
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

	return LoadResult{
		Org:                 org,
		ActiveConfigVersion: version,
		Domain:              domain,
		Config:              cfg,
	}, nil
}
