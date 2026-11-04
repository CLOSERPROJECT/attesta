package living

import (
	"context"
	"fmt"
	"time"
)

// Demo instance ids for Oct-19 smoke / legacy Commit helpers (not SeedDemo stage matrix).
// Truck: YYMMDD-turno-n from truckArrive. Load: client-YYMMDD-n from lots/register.
const (
	DemoTruckInstanceID = "261019-1st-1"
	DemoLoadInstanceID  = "CO-261019-1"
	DemoLineShiftID     = "2026-10-19-1st"
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

func demoSeedCommitter(domains DomainStore, configs ConfigStore) *Committer {
	return &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
}

func commitIntakeThroughLotsClosedCtx(ctx context.Context, c *Committer, org string) (truckID, loadID string, err error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic", "line"}}
	arrive, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value: map[string]any{
			"date": "2026-10-19", "turno": "1st", "truckWeight": 1000.0,
		},
		Mint: startingNewMint(),
	})
	if err != nil {
		return "", "", err
	}
	truckID = arrive.JobInstanceID
	lot, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value: map[string]any{
			"client": "CO", "wasteCode": "160214", "numeroFir": "FIR-1",
			"palletCount": 2.0, "truckWeightAfter": 700.0, "notes": "",
		},
		Mint: spawningNewMint(),
	})
	if err != nil {
		return "", "", err
	}
	for id := range lot.Domain.Kind("load") {
		loadID = id
		break
	}
	if loadID == "" {
		return "", "", fmt.Errorf("living: expected minted load")
	}
	if _, err := c.CloseGroup(ctx, org, op, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
	}); err != nil {
		return "", "", err
	}
	return truckID, loadID, nil
}

func commitIntakeThroughPalletWeight(ctx context.Context, c *Committer, org string) (Domain, error) {
	_, loadID, err := commitIntakeThroughLotsClosedCtx(ctx, c, org)
	if err != nil {
		return Domain{}, err
	}
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic", "line"}}
	line, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": "1st"},
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
		if _, err := c.Commit(ctx, org, op, CommitRequest{
			TaskPath:      GroupTaskPath("line", "palletWeight", ProvisionalGroupInstanceID, "weigh"),
			JobInstanceID: line.JobInstanceID,
			Value:         weighValue(row.id, row.weight),
			Mint:          spawningNewMint(),
		}); err != nil {
			return Domain{}, err
		}
	}
	if _, err := c.CloseGroup(ctx, org, op, CloseGroupRequest{
		JobID: "line", JobInstanceID: line.JobInstanceID, GroupID: "palletWeight",
	}); err != nil {
		return Domain{}, err
	}
	return c.Domains.Get(ctx, org)
}

func commitLineShiftNew(ctx context.Context, c *Committer, org, shift string) (Domain, error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"line"}}
	turno := shift
	if turno == "" {
		turno = "1st"
	}
	result, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": turno},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return Domain{}, err
	}
	return result.Domain, nil
}

// seedLineLoadReadyDomain wipes org Domain then Commits intake lots + line weigh
// so lineLoad readyWhen is true (weighed pallet row ids).
func seedLineLoadReadyDomain(ctx context.Context, domains DomainStore, configs ConfigStore, org string) error {
	if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
		return err
	}
	c := demoSeedCommitter(domains, configs)
	if _, err := commitIntakeThroughPalletWeight(ctx, c, org); err != nil {
		return fmt.Errorf("living seed demo intake: %w", err)
	}
	return nil
}
