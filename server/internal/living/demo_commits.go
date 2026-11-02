package living

import (
	"context"
	"fmt"
	"time"
)

// Demo instance ids for SeedDemo wipe fixture (aligned with living Commit tests).
const (
	DemoIntakeInstanceID = "NIR-261019-NO-1"
	DemoLineShiftID      = "1st"
)

func registerValue() map[string]any {
	return map[string]any{
		"client":             "NO",
		"date":               "2026-10-19",
		"wasteCode":          "191212",
		"idoneitaRiutilizzo": "NIR",
		"truckWeight":        100.0,
		"truckTare":          200.0,
		"palletCount":        2.0,
		"notes":              "",
	}
}

func weighRow(weight float64) map[string]any {
	return map[string]any{
		"weight": weight, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0,
	}
}

func palletWeightValue(weights map[string]any) map[string]any {
	return map[string]any{
		"averagePanelWeight": 10.0,
		"tareKind":           "euro",
		"weights":            weights,
	}
}

func demoSeedCommitter(domains DomainStore, configs ConfigStore) *Committer {
	return &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
}

func commitIntakeThroughCodiceFir(ctx context.Context, c *Committer, org, instanceID string) error {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic"}}
	if _, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
		Mint: startingNewMint(),
	}); err != nil {
		return err
	}
	_, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-1"},
	})
	return err
}

func commitIntakeThroughPalletWeight(ctx context.Context, c *Committer, org, instanceID string) (Domain, error) {
	if err := commitIntakeThroughCodiceFir(ctx, c, org, instanceID); err != nil {
		return Domain{}, err
	}
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"logistic"}}
	result, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: palletWeightValue(map[string]any{
			instanceID + ".1": weighRow(11),
			instanceID + ".2": weighRow(22),
		}),
	})
	if err != nil {
		return Domain{}, err
	}
	return result.Domain, nil
}

func commitLineShiftNew(ctx context.Context, c *Committer, org, shift string) (Domain, error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"line"}}
	result, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: shift,
		Value: map[string]any{"shift": shift},
		Mint:  startingNewMint(),
	})
	if err != nil {
		return Domain{}, err
	}
	return result.Domain, nil
}

// seedLineLoadReadyDomain wipes org Domain then Commits intake through palletWeight
// and a line/shift StartingNew so lineLoad is available with weighed pallet row ids.
func seedLineLoadReadyDomain(ctx context.Context, domains DomainStore, configs ConfigStore, org string) error {
	if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
		return err
	}
	c := demoSeedCommitter(domains, configs)
	if _, err := commitIntakeThroughPalletWeight(ctx, c, org, DemoIntakeInstanceID); err != nil {
		return fmt.Errorf("living seed demo intake: %w", err)
	}
	if _, err := commitLineShiftNew(ctx, c, org, DemoLineShiftID); err != nil {
		return fmt.Errorf("living seed demo line shift: %w", err)
	}
	return nil
}
