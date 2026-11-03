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
	// set this — without it, EmptyDomain() would pick up the stage matrix.
	SkipEmptyDomainStaging bool
}

// Deterministic demo Job instance ids for staged Domain (--replace rebuild).
// Intake ids follow Feragame load grammar from seedRegisterValue (date 2026-11-02).
const (
	SeedIntakeAfterRegisterID = "NIR-261102-NO-1"
	SeedIntakeAfterFirID      = "NIR-261102-SU-1"
	SeedIntakeWeightCheckID   = "NIR-261102-NO-2"
	SeedLineShiftID           = "1st"
)

// SeedDemo puts the demo Feragame Org Config as the Active Config version and,
// when WipeDomain is set or Domain is missing/empty, replaces the org Domain
// with the staged multi-actor fixture (not EmptyDomain), unless
// SkipEmptyDomainStaging opts out of the missing/empty path.
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

// seedStagedDemoDomain wipes then Commits the minimal intake/line stage matrix so
// split-role Worklists diverge (administration / logistic / line).
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

	afterReg, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/register",
		Value:    seedRegisterValue("NO", 2),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if afterReg.JobInstanceID != SeedIntakeAfterRegisterID {
		return fmt.Errorf("living seed: after-register id %q, want %q", afterReg.JobInstanceID, SeedIntakeAfterRegisterID)
	}

	afterFir, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/register",
		Value:    seedRegisterValue("SU", 2),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if afterFir.JobInstanceID != SeedIntakeAfterFirID {
		return fmt.Errorf("living seed: after-fir id %q, want %q", afterFir.JobInstanceID, SeedIntakeAfterFirID)
	}
	if _, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: afterFir.JobInstanceID,
		Value: map[string]any{"codiceFir": "FIR-SEED-FIR"},
	}); err != nil {
		return err
	}

	wcReady, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/register",
		Value:    seedRegisterValue("NO", 2),
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if wcReady.JobInstanceID != SeedIntakeWeightCheckID {
		return fmt.Errorf("living seed: weightCheck id %q, want %q", wcReady.JobInstanceID, SeedIntakeWeightCheckID)
	}
	if _, err := seedCommit(ctx, c, org, admin, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: wcReady.JobInstanceID,
		Value: map[string]any{"codiceFir": "FIR-SEED-WC"},
	}); err != nil {
		return err
	}
	if _, err := seedCommit(ctx, c, org, logistic, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: wcReady.JobInstanceID,
		Value: map[string]any{
			"averagePanelWeight": 10.0,
			"tareKind":           "euro",
			"weights": map[string]any{
				wcReady.JobInstanceID + ".1": map[string]any{
					"weight": 125.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0,
				},
				wcReady.JobInstanceID + ".2": map[string]any{
					"weight": 100.0, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0,
				},
			},
		},
	}); err != nil {
		return err
	}

	shift, err := seedCommit(ctx, c, org, lineOp, CommitRequest{
		TaskPath: "line/shift",
		Value:    map[string]any{"shift": SeedLineShiftID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return err
	}
	if shift.JobInstanceID != SeedLineShiftID {
		return fmt.Errorf("living seed: line shift id %q, want %q", shift.JobInstanceID, SeedLineShiftID)
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

func seedRegisterValue(client string, palletCount float64) map[string]any {
	return map[string]any{
		"client":             client,
		"date":               "2026-11-02",
		"wasteCode":          "191212",
		"idoneitaRiutilizzo": "NIR",
		"truckWeight":        1000.0,
		"truckTare":          200.0,
		"palletCount":        palletCount,
		"notes":              "",
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
