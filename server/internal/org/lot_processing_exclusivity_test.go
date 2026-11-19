package org

import (
	"context"
	"testing"
)

// readyWhenGateAuthorizer mirrors Cerbos living-commit gates without an HTTP PDP:
// deny when sequenceOk or Task readyWhen is false.
type readyWhenGateAuthorizer struct{}

func (readyWhenGateAuthorizer) AllowCommit(_ context.Context, q AuthorizeCommit) (bool, error) {
	if !q.SequenceOk || !q.ReadyWhenOk {
		return false, nil
	}
	if len(q.RolesAllowed) == 0 {
		return true, nil
	}
	for _, have := range q.Operator.Roles {
		for _, want := range q.RolesAllowed {
			if have != "" && have == want {
				return true, nil
			}
		}
	}
	return false, nil
}

func TestIntakeAllowsParallelIncompleteTrucks(t *testing.T) {
	c, _, _ := testTaskCommitter(t, readyWhenGateAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	cfg := mustActiveConfig(t, c)

	emptyWL := projectWorklistUnfiltered(cfg, EmptyDomain(), admin)
	intake, ok := worklistJobByID(emptyWL, "intake")
	if !ok || !intake.CanStart {
		t.Fatalf("intake canStart on empty domain = %#v, want true", intake)
	}

	arrive, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID
	if truckID == "" {
		t.Fatal("empty truck jobInstanceId")
	}
	truck := arrive.Domain.Kind("truck")[truckID]
	if _, has := truck["workingDayId"]; has {
		t.Fatalf("truck must not stamp workingDayId: %#v", truck)
	}
	if _, has := truck["turno"]; has {
		t.Fatalf("truck must not stamp turno: %#v", truck)
	}

	parallel, ok := worklistJobByID(projectWorklistUnfiltered(cfg, arrive.Domain, admin), "intake")
	if !ok || !parallel.CanStart {
		t.Fatalf("intake canStart with incomplete truck = %#v, want true (parallel trucks)", parallel)
	}
	second, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatalf("second truckArrive while prior incomplete must allow: %v", err)
	}
	if second.JobInstanceID == "" || second.JobInstanceID == truckID {
		t.Fatalf("second truck id = %q, want distinct from %q", second.JobInstanceID, truckID)
	}
}

func TestLotExclusivityOneOpenLot(t *testing.T) {
	c, _, _ := testTaskCommitter(t, readyWhenGateAuthorizer{})
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	cfg := mustActiveConfig(t, c)

	_, load1, load2 := plantIntakeTwoLots(t, c)

	first, err := c.Commit(t.Context(), FeragameScope(), logistic, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": load1},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.JobInstanceID != load1 {
		t.Fatalf("startLot instance = %q, want %q", first.JobInstanceID, load1)
	}
	if first.Domain.Kind("lot")[load1]["status"] != "open" {
		t.Fatalf("after startLot = %#v, want open", first.Domain.Kind("lot")[load1])
	}

	lp, ok := worklistJobByID(projectWorklistUnfiltered(cfg, first.Domain, logistic), "lot_processing")
	if !ok || lp.CanStart {
		t.Fatalf("lot_processing canStart with one open lot = %#v, want false", lp)
	}
	_, err = c.Commit(t.Context(), FeragameScope(), logistic, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": load2},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("second startLot while open lot must deny: %v", err)
	}

	plantCommitWeigh(t, c, load1, load1+".1", 11)
	plantCommitWeigh(t, c, load1, load1+".2", 22)
	plantCommitMarkProgress(t, c, load1, map[string]any{
		load1 + ".1": "done",
		load1 + ".2": "done",
	})
	closed := plantCommitCloseLot(t, c, load1)
	if closed.Domain.Kind("lot")[load1]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot = %#v, want fullyProcessed", closed.Domain.Kind("lot")[load1])
	}

	ready, ok := worklistJobByID(projectWorklistUnfiltered(cfg, closed.Domain, logistic), "lot_processing")
	if !ok || !ready.CanStart {
		t.Fatalf("lot_processing canStart after closeLot = %#v, want true", ready)
	}
	second, err := c.Commit(t.Context(), FeragameScope(), logistic, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": load2},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.JobInstanceID != load2 || second.Domain.Kind("lot")[load2]["status"] != "open" {
		t.Fatalf("second startLot = %#v", second.Domain.Kind("lot")[load2])
	}
}

func TestLineCannotStartLotProcessing(t *testing.T) {
	c, _, _ := testTaskCommitter(t, readyWhenGateAuthorizer{})
	_, loadID := plantIntakeOneLot(t, c)
	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	lp, ok := worklistJobByID(projectWorklistUnfiltered(cfg, domain, lineOp), "lot_processing")
	if !ok || lp.CanStart {
		t.Fatalf("line canStart lot_processing = %#v, want false (startLot is logistic)", lp)
	}
	_, err = c.Commit(t.Context(), FeragameScope(), lineOp, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("line startLot must deny: %v", err)
	}
}

func plantCommitArrive(t *testing.T, c *TaskCommitter) string {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID == "" {
		t.Fatal("empty truck jobInstanceId")
	}
	return result.JobInstanceID
}

func plantCommitRegister(t *testing.T, c *TaskCommitter, truckID string, value map[string]any) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         value,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func plantCommitWeightCheck(t *testing.T, c *TaskCommitter, truckID string, truckWeight, lotSum float64) {
	t.Helper()
	if _, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/weightCheck",
		JobInstanceID: truckID,
		Value: map[string]any{
			"checkedTruckWeight": truckWeight,
			"checkedLotSum":      lotSum,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func plantIntakeOneLot(t *testing.T, c *TaskCommitter) (truckID, loadID string) {
	t.Helper()
	truckID = plantCommitArrive(t, c)
	reg := plantCommitRegister(t, c, truckID, registerLotValue())
	loadID = firstLoadID(reg.Domain)
	if loadID == "" {
		t.Fatal("expected minted lot")
	}
	return truckID, loadID
}

func plantIntakeTwoLots(t *testing.T, c *TaskCommitter) (truckID, load1, load2 string) {
	t.Helper()
	truckID = plantCommitArrive(t, c)
	first := plantCommitRegister(t, c, truckID, registerLotValue())
	load1 = firstLoadID(first.Domain)
	if load1 == "" {
		t.Fatal("expected first lot")
	}
	lot2 := registerLotValue()
	lot2["client"] = "DA"
	lot2["numeroFir"] = "FIR-2"
	lot2["truckWeightAfter"] = 400.0
	second := plantCommitRegister(t, c, truckID, lot2)
	for id := range second.Domain.Kind("lot") {
		if id != load1 {
			load2 = id
			break
		}
	}
	if load2 == "" {
		t.Fatal("expected second lot")
	}
	return truckID, load1, load2
}

func plantCommitStartLot(t *testing.T, c *TaskCommitter, lotID string) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": lotID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != lotID {
		t.Fatalf("startLot instance = %q, want %q", result.JobInstanceID, lotID)
	}
	return result
}

func plantCommitWeigh(t *testing.T, c *TaskCommitter, lotID, palletID string, weight float64) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "lot_processing/weigh",
		JobInstanceID: lotID,
		Value:         weighValue(palletID, weight),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func plantCommitMarkProgress(t *testing.T, c *TaskCommitter, lotID string, progress map[string]any) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "lot_processing/markProgress",
		JobInstanceID: lotID,
		Value:         map[string]any{"progress": progress},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func plantCommitCloseLot(t *testing.T, c *TaskCommitter, lotID string) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "lot_processing/closeLot",
		JobInstanceID: lotID,
		Value:         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func plantSeedOpenLotReadyForContainer(t *testing.T, c *TaskCommitter) (loadID string) {
	t.Helper()
	_, loadID = plantIntakeOneLot(t, c)
	_ = plantCommitStartLot(t, c, loadID)
	_ = plantCommitWeigh(t, c, loadID, loadID+".1", 11)
	_ = plantCommitWeigh(t, c, loadID, loadID+".2", 22)
	return loadID
}
