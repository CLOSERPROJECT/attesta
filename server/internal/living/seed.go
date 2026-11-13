package living

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SeedScope selects what SeedDemo writes. Zero value is SeedScopeBoth.
type SeedScope int

const (
	// SeedScopeBoth puts Active Config then stages Domain per WipeDomain /
	// empty-Domain rules (prior SeedDemo behavior).
	SeedScopeBoth SeedScope = iota
	// SeedScopeFeragameConfig puts Active Config and always wipes Domain
	// (+ passports) to empty; never stages.
	SeedScopeFeragameConfig
	// SeedScopeFeragameDomain puts Active Config and always wipes+stages Domain.
	SeedScopeFeragameDomain
	// SeedScopeFeragameIdentity is living-identical to Config (empty wipe);
	// cmd/server pairs it with Feragame Appwrite hard reset.
	SeedScopeFeragameIdentity
	// SeedScopeFeragameFull is living-identical to Domain (wipe+stage);
	// cmd/server pairs it with Feragame Appwrite hard reset.
	SeedScopeFeragameFull
)

// SeedOptions controls Domain wipe and empty-Domain staging during SeedDemo.
// Only task seed / tests should call SeedDemo; WipeDomain follows Seed Replace —
// never ungated in prod.
//
// Staging runs when WipeDomain is set, or when Domain is missing or empty
// (no JobInstances and no Submissions). SkipEmptyDomainStaging opts out of that
// missing/empty path so callers can keep Active Config without staged data.
// Scope narrows the write surface; see SeedScope.
type SeedOptions struct {
	WipeDomain bool
	// SkipEmptyDomainStaging keeps a missing or empty Domain untouched when
	// WipeDomain is false. Handler/smoke helpers that only want Active Config
	// set this — without it, EmptyDomain() would pick up the focused fixture.
	SkipEmptyDomainStaging bool
	// Passports is wiped for the org when Domain is rebuilt (seed --replace /
	// Feragame scopes). Optional in tests that never lock; task seed must wire it.
	Passports PassportStore
	// Scope selects Feragame config/domain/identity/full or both (default).
	Scope SeedScope
}

// Deterministic demo Job instance ids for staged Domain (--replace rebuild).
// Truck grammar: YYMMDD-n from open workingDay date (2026-11-02 → 261102-1).
// Line grammar: date-turno from openTurno.
// Lot grammar: client-YYMMDD-n from intake/register.
const (
	SeedTruckID  = "261102-1"
	SeedLineID   = "2026-11-02-1st"
	SeedLoadCOID = "CO-261102-1"
	SeedLoadDAID = "DA-261102-1"
)

// SeedDemo applies the demo Feragame fixture per opts.Scope:
//   - SeedScopeBoth (default): Active Config, then Domain when WipeDomain is set
//     or Domain is missing/empty (unless SkipEmptyDomainStaging).
//   - SeedScopeFeragameConfig / Identity: Active Config + always wipe Domain
//     (+ passports) to empty; never stages.
//   - SeedScopeFeragameDomain / Full: Active Config + always wipe + stage.
//
// Demo/fixture only — product hosts must not require this demo seed path.
// There is no harness /api/runs snapshot PUT — reset is this re-seed path.
func SeedDemo(ctx context.Context, domains DomainStore, configs ConfigStore, opts SeedOptions) (OrgConfig, error) {
	if configs == nil {
		return OrgConfig{}, fmt.Errorf("living: config store required")
	}
	switch opts.Scope {
	case SeedScopeFeragameConfig, SeedScopeFeragameIdentity:
		return seedDemoFeragameEmptyDomain(ctx, domains, configs, opts.Passports)
	case SeedScopeFeragameDomain, SeedScopeFeragameFull:
		return seedDemoBoth(ctx, domains, configs, SeedOptions{
			WipeDomain: true,
			Passports:  opts.Passports,
		})
	default:
		return seedDemoBoth(ctx, domains, configs, opts)
	}
}

func seedDemoBoth(ctx context.Context, domains DomainStore, configs ConfigStore, opts SeedOptions) (OrgConfig, error) {
	cfg, err := SeedFixtureConfigStore(configs)
	if err != nil {
		return OrgConfig{}, err
	}
	needDomain := opts.WipeDomain
	if !needDomain {
		if domains == nil {
			return cfg, nil
		}
		domain, err := domains.Get(ctx, cfg.Org)
		switch {
		case err == nil:
			if opts.SkipEmptyDomainStaging || !domainSeedEmpty(domain) {
				return cfg, nil
			}
			needDomain = true
		case errors.Is(err, ErrNotFound):
			if opts.SkipEmptyDomainStaging {
				return cfg, nil
			}
			needDomain = true
		default:
			return OrgConfig{}, err
		}
	}
	if domains == nil {
		return OrgConfig{}, fmt.Errorf("living: domain store required to seed Domain")
	}
	if err := seedStagedDemoDomain(ctx, domains, configs, opts.Passports, cfg.Org); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

func seedDemoFeragameEmptyDomain(ctx context.Context, domains DomainStore, configs ConfigStore, passports PassportStore) (OrgConfig, error) {
	cfg, err := SeedFixtureConfigStore(configs)
	if err != nil {
		return OrgConfig{}, err
	}
	if domains == nil {
		return OrgConfig{}, fmt.Errorf("living: domain store required for Feragame empty-Domain seed")
	}
	if passports != nil {
		if err := passports.DeleteByOrg(ctx, cfg.Org); err != nil {
			return OrgConfig{}, err
		}
	}
	if err := domains.Put(ctx, cfg.Org, EmptyDomain()); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

// domainSeedEmpty is true when Domain has no JobInstances and no Submissions
// (entities alone do not count as staged demo data).
func domainSeedEmpty(d Domain) bool {
	return len(d.JobInstances) == 0 && len(d.Submissions) == 0
}

// seedStagedDemoDomain wipes then Commits one coherent plant story:
// one intake with two lots (CO + DA), one line turn that startLot→weigh→markProgress→
// containerReg→closeLot for each lot (sole fullyProcessed writer), for Passport.
// Leaves weightCheck open; Passport Can start is ready for both lots.
// When passports is set, org rows are deleted before Domain Put so deterministic
// seed subject ids can Lock again after --replace / --feragame-domain / --feragame-full.
func seedStagedDemoDomain(ctx context.Context, domains DomainStore, configs ConfigStore, passports PassportStore, org string) error {
	if passports != nil {
		if err := passports.DeleteByOrg(ctx, org); err != nil {
			return err
		}
	}
	if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
		return err
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) },
	}
	admin := OperatorIdentity{UserID: "seed-admin", Org: org, Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "seed-logistic", Org: org, Roles: []string{"logistic"}}
	lineOp := OperatorIdentity{UserID: "seed-line", Org: org, Roles: []string{"line"}}

	if _, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "workingDay/open",
		Value:    map[string]any{"date": "2026-11-02", "mode": "multi"},
		Mint:     startingNewMint(),
	}); err != nil {
		return err
	}

	arrive, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    seedTruckArriveValue(1000),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if arrive.JobInstanceID != SeedTruckID {
		return fmt.Errorf("living seed: truck id %q, want %q", arrive.JobInstanceID, SeedTruckID)
	}

	for _, lot := range []struct {
		client, fir string
		pallets     float64
		after       float64
		wantLot     string
	}{
		{"CO", "FIR-SEED-CO", 2, 700, SeedLoadCOID},
		{"DA", "FIR-SEED-DA", 2, 400, SeedLoadDAID},
	} {
		res, err := seedCommit(ctx, c, org, admin, CommitRequest{
			TaskPath:      "intake/register",
			JobInstanceID: SeedTruckID,
			Value:         seedLotRegisterValue(lot.client, lot.fir, lot.pallets, lot.after),
		})
		if err != nil {
			return err
		}
		if _, ok := res.Domain.Kind("lot")[lot.wantLot]; !ok {
			return fmt.Errorf("living seed: expected lot %q, got %#v", lot.wantLot, res.Domain.Kind("lot"))
		}
	}

	line, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-11-02", "turno": "1st"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if line.JobInstanceID != SeedLineID {
		return fmt.Errorf("living seed: line id %q, want %q", line.JobInstanceID, SeedLineID)
	}
	for _, lot := range []struct {
		id string
		pallets []struct {
			id     string
			weight float64
		}
	}{
		{SeedLoadCOID, []struct {
			id     string
			weight float64
		}{{SeedLoadCOID + ".1", 125.0}, {SeedLoadCOID + ".2", 100.0}}},
		{SeedLoadDAID, []struct {
			id     string
			weight float64
		}{{SeedLoadDAID + ".1", 90.0}, {SeedLoadDAID + ".2", 85.0}}},
	} {
		if _, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
			TaskPath:      "line/startLot",
			JobInstanceID: SeedLineID,
			Value:         map[string]any{"lotId": lot.id},
		}); err != nil {
			return err
		}
		progress := map[string]any{}
		for _, row := range lot.pallets {
			if _, err := seedCommit(ctx, c, org, logistic, CommitRequest{
				TaskPath:      "line/weigh",
				JobInstanceID: SeedLineID,
				Value:         weighValue(row.id, row.weight),
			}); err != nil {
				return err
			}
			progress[row.id] = "done"
		}
		if _, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
			TaskPath:      "line/markProgress",
			JobInstanceID: SeedLineID,
			Value:         map[string]any{"progress": progress},
		}); err != nil {
			return err
		}
		if _, err := seedCommit(ctx, c, org, logistic, CommitRequest{
			TaskPath:      "line/containerReg",
			JobInstanceID: SeedLineID,
			Value: map[string]any{
				"sourceLotId": lot.id,
				"polvereDiVetro": []any{
					map[string]any{"weight": 100.0},
				},
			},
		}); err != nil {
			return err
		}
		if _, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
			TaskPath:      "line/closeLot",
			JobInstanceID: SeedLineID,
			Value:         map[string]any{},
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedCommit(ctx context.Context, c *Committer, org string, op OperatorIdentity, req CommitRequest) (CommitResult, error) {
	res, err := c.Commit(ctx, org, op, req)
	if err != nil {
		return CommitResult{}, fmt.Errorf("living seed %s: %w", req.TaskPath, err)
	}
	return res, nil
}

func seedTruckArriveValue(truckWeight float64) map[string]any {
	return map[string]any{
		"truckWeight": truckWeight,
	}
}

func seedLotRegisterValue(client, numeroFir string, palletCount, truckWeightAfter float64) map[string]any {
	return map[string]any{
		"client":           client,
		"wasteCode":        "160214",
		"numeroFir":        numeroFir,
		"palletCount":      palletCount,
		"truckWeightAfter": truckWeightAfter,
		"notes":            "",
	}
}

// FixtureConfigRoles returns the Feragame Config role catalog (demo all-roles membership).
func FixtureConfigRoles() ([]string, error) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), cfg.Roles...), nil
}
