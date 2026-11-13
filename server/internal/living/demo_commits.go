package living

import (
	"context"
	"fmt"
	"time"
)

// Demo instance ids for Oct-19 smoke / legacy Commit helpers (not SeedDemo stage matrix).
// Truck: YYMMDD-n from open workingDay date. Lot: client-YYMMDD-n from intake/register.
const (
	DemoTruckInstanceID = "261019-1"
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

func commitOpenWorkingDayCtx(ctx context.Context, c *Committer, org, date, mode string) (string, error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic", "line"}}
	if date == "" {
		date = "2026-10-19"
	}
	if mode == "" {
		mode = "multi"
	}
	if domain, err := c.Domains.Get(ctx, org); err == nil {
		if day, ok := domain.Kind("workingDay")[date]; ok && day["status"] == "open" {
			return date, nil
		}
	}
	result, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "workingDay/open",
		Value:    map[string]any{"date": date, "mode": mode},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return "", err
	}
	return result.JobInstanceID, nil
}

func commitIntakeThroughLotsClosedCtx(ctx context.Context, c *Committer, org string) (truckID, loadID string, err error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic", "line"}}
	if _, err := commitOpenWorkingDayCtx(ctx, c, org, "2026-10-19", "multi"); err != nil {
		return "", "", err
	}
	arrive, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    map[string]any{"truckWeight": 1000.0},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return "", "", err
	}
	truckID = arrive.JobInstanceID
	lot, err := c.Commit(ctx, org, op, CommitRequest{
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
		return "", "", fmt.Errorf("living: expected minted lot")
	}
	return truckID, loadID, nil
}

func commitIntakeThroughPalletWeight(ctx context.Context, c *Committer, org string) (Domain, error) {
	_, loadID, err := commitIntakeThroughLotsClosedCtx(ctx, c, org)
	if err != nil {
		return Domain{}, err
	}
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "logistic", "line"}}
	if _, err := commitOpenWorkingDayCtx(ctx, c, org, "2026-10-19", "multi"); err != nil {
		return Domain{}, err
	}
	line, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": "1st"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return Domain{}, err
	}
	if _, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath:      "line/startLot",
		JobInstanceID: line.JobInstanceID,
		Value:         map[string]any{"lotId": loadID},
	}); err != nil {
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
			TaskPath:      "line/weigh",
			JobInstanceID: line.JobInstanceID,
			Value:         weighValue(row.id, row.weight),
		}); err != nil {
			return Domain{}, err
		}
	}
	return c.Domains.Get(ctx, org)
}

func commitLineShiftNew(ctx context.Context, c *Committer, org, shift string) (Domain, error) {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "line"}}
	turno := shift
	if turno == "" {
		turno = "1st"
	}
	if _, err := commitOpenWorkingDayCtx(ctx, c, org, "2026-10-19", "multi"); err != nil {
		return Domain{}, err
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

func commitCloseTurnoCtx(ctx context.Context, c *Committer, org, lineID string) error {
	op := OperatorIdentity{UserID: "ada", Org: org, Roles: []string{"administration", "line"}}
	_, err := c.Commit(ctx, org, op, CommitRequest{
		TaskPath:      "line/closeTurno",
		JobInstanceID: lineID,
		Value:         map[string]any{},
	})
	return err
}

// seedLineLoadReadyDomain wipes org Domain then Commits intake + openTurno + startLot +
// weigh so markProgress / containerReg see an open lot with weighed pallets.
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
