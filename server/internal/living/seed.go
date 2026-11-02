package living

import (
	"context"
	"fmt"
)

// SeedOptions controls Domain wipe during SeedDemo. Only task seed / tests
// should call SeedDemo; WipeDomain follows Seed Replace — never ungated in prod.
type SeedOptions struct {
	WipeDomain bool
}

// SeedDemo puts the demo Feragame Org Config as the Active Config version and,
// when WipeDomain is set, replaces the org Domain with a lineLoad-ready demo
// (intake through palletWeight + line/shift StartingNew).
// Demo/fixture only — product hosts must not require this demo seed path.
// There is no harness /api/runs snapshot PUT — reset is this re-seed path.
func SeedDemo(ctx context.Context, domains DomainStore, configs ConfigStore, opts SeedOptions) (OrgConfig, error) {
	if configs == nil {
		return OrgConfig{}, fmt.Errorf("living: config store required")
	}
	cfg, err := SeedFixtureConfigStore(configs)
	if err != nil {
		return OrgConfig{}, err
	}
	if !opts.WipeDomain {
		return cfg, nil
	}
	if domains == nil {
		return OrgConfig{}, fmt.Errorf("living: domain store required to wipe")
	}
	if err := seedLineLoadReadyDomain(ctx, domains, configs, cfg.Org); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

// FixtureConfigRoles returns the Feragame Config role catalog (demo all-roles membership).
func FixtureConfigRoles() ([]string, error) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), cfg.Roles...), nil
}
