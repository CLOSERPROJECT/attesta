package living

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SeedOptions controls Domain wipe and empty-Domain staging during SeedDemo.
// Only task seed / tests should call SeedDemo; WipeDomain follows Seed Replace —
// never ungated in prod.
//
// Staging runs when WipeDomain is set, or when Domain is missing or empty
// (no JobInstances and no Submissions). SkipEmptyDomainStaging opts out of that
// missing/empty path so callers can keep Active Config without staged data.
type SeedOptions struct {
	WipeDomain bool
	// SkipEmptyDomainStaging keeps a missing or empty Domain untouched when
	// WipeDomain is false. Handler/smoke helpers that only want Active Config
	// set this — without it, EmptyDomain() would pick up the focused fixture.
	SkipEmptyDomainStaging bool
}

// Deterministic demo Job instance ids for staged Domain (--replace rebuild).
// Truck grammar: YYMMDD-turno-n from seedTruckArriveValue (date 2026-11-02).
// Line grammar: date-turno from openTurno.
// Load grammar: client-YYMMDD-n from lots/register.
const (
	SeedTruckID  = "261102-1st-1"
	SeedLineID   = "2026-11-02-1st"
	SeedLoadNOID = "NO-261102-1"
	SeedLoadSUID = "SU-261102-1"
)

// SeedDemo puts the demo Feragame Org Config as the Active Config version and,
// when WipeDomain is set or Domain is missing/empty, replaces the org Domain
// with the focused plant fixture (one intake × two loads + one line turn),
// unless SkipEmptyDomainStaging opts out of the missing/empty path.
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
	if err := seedStagedDemoDomain(ctx, domains, configs, cfg.Org); err != nil {
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
// one intake with two loads (NO + SU), one line turn with all pallets weighed,
// lineLoad done, and both loads register-closed (fullyProcessed) for Passport.
// Leaves weightCheck open; Passport Can start is ready for both loads.
func seedStagedDemoDomain(ctx context.Context, domains DomainStore, configs ConfigStore, org string) error {
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

	arrive, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    seedTruckArriveValue("1st", 1000),
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
		wantLoad    string
	}{
		{"NO", "FIR-SEED-NO", 2, 700, SeedLoadNOID},
		{"SU", "FIR-SEED-SU", 2, 400, SeedLoadSUID},
	} {
		res, err := seedCommit(ctx, c, org, admin, CommitRequest{
			TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
			JobInstanceID: SeedTruckID,
			Value:         seedLotRegisterValue(lot.client, lot.fir, lot.pallets, lot.after),
			Mint:          spawningNewMint(),
		})
		if err != nil {
			return err
		}
		if _, ok := res.Domain.Kind("load")[lot.wantLoad]; !ok {
			return fmt.Errorf("living seed: expected load %q, got %#v", lot.wantLoad, res.Domain.Kind("load"))
		}
	}
	if _, err := c.CloseGroup(ctx, org, admin, CloseGroupRequest{
		JobID: "intake", JobInstanceID: SeedTruckID, GroupID: "lots",
	}); err != nil {
		return fmt.Errorf("living seed close lots: %w", err)
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
	for _, row := range []struct {
		id     string
		weight float64
	}{
		{SeedLoadNOID + ".1", 125.0},
		{SeedLoadNOID + ".2", 100.0},
		{SeedLoadSUID + ".1", 90.0},
		{SeedLoadSUID + ".2", 85.0},
	} {
		if _, err := seedCommit(ctx, c, org, logistic, CommitRequest{
			TaskPath:      GroupTaskPath("line", "palletWeight", ProvisionalGroupInstanceID, "weigh"),
			JobInstanceID: SeedLineID,
			Value:         weighValue(row.id, row.weight),
			Mint:          spawningNewMint(),
		}); err != nil {
			return err
		}
	}
	if _, err := c.CloseGroup(ctx, org, logistic, CloseGroupRequest{
		JobID: "line", JobInstanceID: SeedLineID, GroupID: "palletWeight",
	}); err != nil {
		return fmt.Errorf("living seed close palletWeight: %w", err)
	}

	if _, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
		TaskPath:      "line/lineLoad",
		JobInstanceID: SeedLineID,
		Value: map[string]any{
			"progress": map[string]any{
				SeedLoadNOID + ".1": "done",
				SeedLoadNOID + ".2": "done",
				SeedLoadSUID + ".1": "done",
				SeedLoadSUID + ".2": "done",
			},
		},
	}); err != nil {
		return err
	}
	for _, loadID := range []string{SeedLoadNOID, SeedLoadSUID} {
		if _, err := seedCommit(ctx, c, org, logistic, CommitRequest{
			TaskPath:      GroupTaskPath("line", "containerReg", ProvisionalGroupInstanceID, "register"),
			JobInstanceID: SeedLineID,
			Value: map[string]any{
				"sourceLoadId": loadID,
				"polvereDiVetro": []any{
					map[string]any{"weight": 100.0},
				},
			},
			Mint: spawningNewMint(),
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

func seedTruckArriveValue(turno string, truckWeight float64) map[string]any {
	return map[string]any{
		"date":        "2026-11-02",
		"turno":       turno,
		"truckWeight": truckWeight,
	}
}

func seedLotRegisterValue(client, numeroFir string, palletCount, truckWeightAfter float64) map[string]any {
	return map[string]any{
		"client":           client,
		"wasteCode":        "191212",
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
