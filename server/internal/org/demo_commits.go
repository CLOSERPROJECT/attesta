package org

import (
	"context"
	"fmt"
	"time"
)

// Demo instance ids for smoke / legacy Commit helpers (not SeedDemo stage matrix).
// Truck / lot grammars use $today from demoSeedCommitter (2026-10-19 → 261019).
// lot_processing Job instance id = lot id (DemoLoadInstanceID after startLot).
const (
	DemoTruckInstanceID = "261019-1"
	DemoLoadInstanceID  = "CO-261019-1"
)

func weighValue(palletID string, weight float64) map[string]any {
	return map[string]any{
		"palletId":           palletID,
		"averagePanelWeight": 10.0,
		"tareKind":           "euro",
		"weight":             weight,
		"idoneita":           "NIR",
		"panelCount":         1.0,
	}
}

func demoSeedCommitter(domains DomainStore, configs ConfigStore) *TaskCommitter {
	return &TaskCommitter{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC) },
	}
}

func commitIntakeThroughLotsClosedCtx(ctx context.Context, c *TaskCommitter, scope Scope) (truckID, loadID string, err error) {
	op := OperatorIdentity{UserID: "ada", Org: scope.Org, Roles: []string{"administration", "logistic", "line"}}
	arrive, err := c.Commit(ctx, scope, op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    map[string]any{"truckWeight": 1000.0},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return "", "", err
	}
	truckID = arrive.JobInstanceID
	lot, err := c.Commit(ctx, scope, op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value: map[string]any{
			"client": "CO", "wasteCode": "160214", "numeroFir": "FIR-1",
			"palletCount": 2.0, "truckWeightAfter": 700.0, "notes": "",
		},
	})
	if err != nil {
		return "", "", err
	}
	for id := range lot.Domain.Kind("lot") {
		loadID = id
		break
	}
	if loadID == "" {
		return "", "", fmt.Errorf("org: expected minted lot")
	}
	return truckID, loadID, nil
}

func commitIntakeThroughPalletWeight(ctx context.Context, c *TaskCommitter, scope Scope) (Domain, error) {
	_, loadID, err := commitIntakeThroughLotsClosedCtx(ctx, c, scope)
	if err != nil {
		return Domain{}, err
	}
	op := OperatorIdentity{UserID: "ada", Org: scope.Org, Roles: []string{"administration", "logistic", "line"}}
	started, err := c.Commit(ctx, scope, op, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return Domain{}, err
	}
	for _, row := range []struct {
		id     string
		weight float64
	}{
		{loadID + ".1", 11},
		{loadID + ".2", 22},
	} {
		if _, err := c.Commit(ctx, scope, op, CommitRequest{
			TaskPath:      "lot_processing/weigh",
			JobInstanceID: started.JobInstanceID,
			Value:         weighValue(row.id, row.weight),
		}); err != nil {
			return Domain{}, err
		}
	}
	return c.Domains.Get(ctx, scope)
}

// seedLotProcessingReadyDomain wipes stream Domain then Commits intake + startLot + weigh
// so markProgress / containerReg see an open lot with weighed pallets.
func seedLotProcessingReadyDomain(ctx context.Context, domains DomainStore, configs ConfigStore, scope Scope) error {
	if err := domains.Put(ctx, scope, EmptyDomain()); err != nil {
		return err
	}
	c := demoSeedCommitter(domains, configs)
	if _, err := commitIntakeThroughPalletWeight(ctx, c, scope); err != nil {
		return fmt.Errorf("org seed demo intake: %w", err)
	}
	return nil
}
