package org

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
	// SeedScopeFeragameConfig puts Active Config and (unless KeepDomain) wipes
	// Domain (+ passports) to empty; never stages.
	SeedScopeFeragameConfig
	// SeedScopeFeragameDomain puts Active Config and always wipes+stages Domain.
	SeedScopeFeragameDomain
	// SeedScopeFeragameIdentity is identical to Config;
	// cmd/server pairs it with Feragame Appwrite hard reset.
	SeedScopeFeragameIdentity
	// SeedScopeFeragameFull is identical to Domain (wipe+stage);
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
	// WipeDomain is false. API/smoke helpers that only want Active Config
	// set this — without it, EmptyDomain() would pick up the focused fixture.
	SkipEmptyDomainStaging bool
	// DPPs is wiped for the org when Domain is rebuilt (seed --replace /
	// Feragame scopes). Optional in tests that never lock; task seed must wire it.
	DPPs DPPStore
	// KeepDomain is valid only with Feragame Config/Identity: put Active Config
	// and leave Domain (+ passports) untouched. Rejected for Domain/Full.
	KeepDomain bool
	// Scope selects Feragame config/domain/identity/full or both (default).
	Scope SeedScope
}

// Deterministic demo Job instance ids for staged Domain (--replace rebuild).
// Truck / lot grammars use $today from the seed clock (2026-11-02 → 261102).
// lot_processing Job instance id = lot id (1:1); no day/turno instances.
//
// Parallel-truck demo matrix (§9 / #315):
//
//	Truck A (SeedTruckID): intake checked; SeedLoadCOID finished+DPP; SeedLoadDAID open with some pallets weighed.
//	Truck B (SeedTruckBID): intake with SeedTruckBLotID registered, weightCheck still open.
const (
	SeedTruckID     = "261102-1"
	SeedTruckBID    = "261102-2"
	SeedLoadCOID    = "CO-261102-1"
	SeedLoadDAID    = "DA-261102-1"
	SeedTruckBLotID = "DI-261102-1"
)

// SeedDemo applies the demo Feragame fixture per opts.Scope:
//   - SeedScopeBoth (default): Active Config, then Domain when WipeDomain is set
//     or Domain is missing/empty (unless SkipEmptyDomainStaging).
//   - SeedScopeFeragameConfig / Identity: Active Config; wipe Domain to empty
//     unless KeepDomain; never stages.
//   - SeedScopeFeragameDomain / Full: Active Config + always wipe + stage
//     (KeepDomain is invalid).
//
// Demo/fixture only — product hosts must not require this demo seed path.
// There is no harness /api/runs snapshot PUT — reset is this re-seed path.
func SeedDemo(ctx context.Context, domains DomainStore, configs ConfigStore, opts SeedOptions) (OrgConfig, error) {
	if configs == nil {
		return OrgConfig{}, fmt.Errorf("org: config store required")
	}
	switch opts.Scope {
	case SeedScopeFeragameConfig, SeedScopeFeragameIdentity:
		if opts.KeepDomain {
			return SeedFixtureConfigStore(configs)
		}
		return seedDemoFeragameEmptyDomain(ctx, domains, configs, opts.DPPs)
	case SeedScopeFeragameDomain, SeedScopeFeragameFull:
		if opts.KeepDomain {
			return OrgConfig{}, fmt.Errorf("org: KeepDomain is only valid with Feragame config/identity scopes")
		}
		return seedDemoBoth(ctx, domains, configs, SeedOptions{
			WipeDomain: true,
			DPPs:       opts.DPPs,
		})
	default:
		if opts.KeepDomain {
			return OrgConfig{}, fmt.Errorf("org: KeepDomain is only valid with Feragame config/identity scopes")
		}
		return seedDemoBoth(ctx, domains, configs, opts)
	}
}

func seedDemoBoth(ctx context.Context, domains DomainStore, configs ConfigStore, opts SeedOptions) (OrgConfig, error) {
	cfg, err := SeedFixtureConfigStore(configs)
	if err != nil {
		return OrgConfig{}, err
	}
	scope, err := ScopeFromConfig(cfg)
	if err != nil {
		return OrgConfig{}, err
	}
	needDomain := opts.WipeDomain
	if !needDomain {
		if domains == nil {
			return cfg, nil
		}
		domain, err := domains.Get(ctx, scope)
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
		return OrgConfig{}, fmt.Errorf("org: domain store required to seed Domain")
	}
	if err := seedStagedDemoDomain(ctx, domains, configs, opts.DPPs, scope); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

func seedDemoFeragameEmptyDomain(ctx context.Context, domains DomainStore, configs ConfigStore, passports DPPStore) (OrgConfig, error) {
	cfg, err := SeedFixtureConfigStore(configs)
	if err != nil {
		return OrgConfig{}, err
	}
	scope, err := ScopeFromConfig(cfg)
	if err != nil {
		return OrgConfig{}, err
	}
	if domains == nil {
		return OrgConfig{}, fmt.Errorf("org: domain store required for Feragame empty-Domain seed")
	}
	if passports != nil {
		if err := passports.DeleteByOrg(ctx, scope.Org); err != nil {
			return OrgConfig{}, err
		}
	}
	if err := domains.Put(ctx, scope, EmptyDomain()); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

// domainSeedEmpty is true when Domain has no JobInstances and no Submissions
// (entities alone do not count as staged demo data).
func domainSeedEmpty(d Domain) bool {
	return len(d.JobInstances) == 0 && len(d.Submissions) == 0
}

// seedStagedDemoDomain wipes then Commits the parallel-truck demo matrix:
//
//	Truck A — weightCheck done; CO finished through lot_processing/lock (DPP);
//	          DA open on the line with one pallet weighed.
//	Truck B — arrive + register only (weightCheck open) so /my shows two active trucks.
//
// When passports is set, org rows are deleted before Domain Put so deterministic
// seed subject ids can Lock again after --replace / --feragame-domain / --feragame-full.
// A nil passports store still stages Domain facts via an ephemeral DPPStore for lock.
func seedStagedDemoDomain(ctx context.Context, domains DomainStore, configs ConfigStore, passports DPPStore, scope Scope) error {
	scope, err := scope.Normalize()
	if err != nil {
		return err
	}
	org := scope.Org
	if passports != nil {
		if err := passports.DeleteByOrg(ctx, org); err != nil {
			return err
		}
	}
	if err := domains.Put(ctx, scope, EmptyDomain()); err != nil {
		return err
	}
	dppStore := passports
	if dppStore == nil {
		dppStore = NewMemoryDPPStore()
	}
	c := &TaskCommitter{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		DPPs:       dppStore,
		Now:        func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) },
	}
	admin := OperatorIdentity{UserID: "seed-admin", Org: org, Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "seed-logistic", Org: org, Roles: []string{"logistic"}}
	lineOp := OperatorIdentity{UserID: "seed-line", Org: org, Roles: []string{"line"}}

	arriveA, err := seedCommit(ctx, c, scope, admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    seedTruckArriveValue(1000),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if arriveA.JobInstanceID != SeedTruckID {
		return fmt.Errorf("org seed: truck A id %q, want %q", arriveA.JobInstanceID, SeedTruckID)
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
		res, err := seedCommit(ctx, c, scope, admin, CommitRequest{
			TaskPath:      "intake/register",
			JobInstanceID: SeedTruckID,
			Value:         seedLotRegisterValue(lot.client, lot.fir, lot.pallets, lot.after),
		})
		if err != nil {
			return err
		}
		if _, ok := res.Domain.Kind("lot")[lot.wantLot]; !ok {
			return fmt.Errorf("org seed: expected lot %q, got %#v", lot.wantLot, res.Domain.Kind("lot"))
		}
	}

	if _, err := seedCommit(ctx, c, scope, admin, CommitRequest{
		TaskPath:      "intake/weightCheck",
		JobInstanceID: SeedTruckID,
		Value: map[string]any{
			"checkedTruckWeight": 1000.0,
			"checkedLotSum":      600.0,
		},
	}); err != nil {
		return err
	}

	if err := seedFinishLotWithDPP(ctx, c, scope, logistic, lineOp, admin, SeedLoadCOID, []struct {
		id     string
		weight float64
	}{{SeedLoadCOID + ".1", 125.0}, {SeedLoadCOID + ".2", 100.0}}); err != nil {
		return err
	}

	startedDA, err := seedCommit(ctx, c, scope, logistic, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": SeedLoadDAID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if startedDA.JobInstanceID != SeedLoadDAID {
		return fmt.Errorf("org seed: lot_processing id %q, want %q", startedDA.JobInstanceID, SeedLoadDAID)
	}
	if _, err := seedCommit(ctx, c, scope, lineOp, CommitRequest{
		TaskPath:      "lot_processing/weigh",
		JobInstanceID: SeedLoadDAID,
		Value:         weighValue(SeedLoadDAID+".1", 90.0),
	}); err != nil {
		return err
	}

	arriveB, err := seedCommit(ctx, c, scope, admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    seedTruckArriveValue(1000),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if arriveB.JobInstanceID != SeedTruckBID {
		return fmt.Errorf("org seed: truck B id %q, want %q", arriveB.JobInstanceID, SeedTruckBID)
	}
	resB, err := seedCommit(ctx, c, scope, admin, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: SeedTruckBID,
		Value:         seedLotRegisterValue("DI", "FIR-SEED-B-DI", 2, 700),
	})
	if err != nil {
		return err
	}
	if _, ok := resB.Domain.Kind("lot")[SeedTruckBLotID]; !ok {
		return fmt.Errorf("org seed: expected lot %q, got %#v", SeedTruckBLotID, resB.Domain.Kind("lot"))
	}
	return nil
}

func seedFinishLotWithDPP(ctx context.Context, c *TaskCommitter, scope Scope, logistic, lineOp, admin OperatorIdentity, lotID string, pallets []struct {
	id     string
	weight float64
}) error {
	started, err := seedCommit(ctx, c, scope, logistic, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": lotID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if started.JobInstanceID != lotID {
		return fmt.Errorf("org seed: lot_processing id %q, want %q", started.JobInstanceID, lotID)
	}
	progress := map[string]any{}
	for _, row := range pallets {
		if _, err := seedCommit(ctx, c, scope, lineOp, CommitRequest{
			TaskPath:      "lot_processing/weigh",
			JobInstanceID: lotID,
			Value:         weighValue(row.id, row.weight),
		}); err != nil {
			return err
		}
		progress[row.id] = "done"
	}
	if _, err := seedCommit(ctx, c, scope, lineOp, CommitRequest{
		TaskPath:      "lot_processing/markProgress",
		JobInstanceID: lotID,
		Value:         map[string]any{"progress": progress},
	}); err != nil {
		return err
	}
	if _, err := seedCommit(ctx, c, scope, logistic, CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: lotID,
		Value: map[string]any{
			"sourceLotId": lotID,
			"polvereDiVetro": []any{
				map[string]any{"weight": 100.0},
			},
		},
	}); err != nil {
		return err
	}
	if _, err := seedCommit(ctx, c, scope, logistic, CommitRequest{
		TaskPath:      "lot_processing/closeLot",
		JobInstanceID: lotID,
		Value:         map[string]any{},
	}); err != nil {
		return err
	}
	if _, err := seedCommit(ctx, c, scope, admin, CommitRequest{
		TaskPath:      "lot_processing/lock",
		JobInstanceID: lotID,
		Value:         map[string]any{"ack": true},
	}); err != nil {
		return err
	}
	return nil
}

func seedCommit(ctx context.Context, c *TaskCommitter, scope Scope, op OperatorIdentity, req CommitRequest) (CommitResult, error) {
	res, err := c.Commit(ctx, scope, op, req)
	if err != nil {
		return CommitResult{}, fmt.Errorf("org seed %s: %w", req.TaskPath, err)
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
