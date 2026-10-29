package living

import (
	"context"
	"fmt"
)

// SeedOptions controls Domain wipe during SeedDemo. Hosts must gate WipeDomain
// (for example ATTESTA_LIVING_SEED); never enable wipe ungated in production.
type SeedOptions struct {
	WipeDomain bool
}

// SeedDemo puts the truck-loop Org Config as the Active Config version and,
// when WipeDomain is set, replaces the org Domain with EmptyDomain.
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
	if err := domains.Put(ctx, cfg.Org, EmptyDomain()); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

// FixtureConfigRoles returns the truck-loop Config role catalog (demo all-roles membership).
func FixtureConfigRoles() ([]string, error) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), cfg.Roles...), nil
}
