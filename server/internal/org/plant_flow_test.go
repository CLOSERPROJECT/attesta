package org

import (
	"testing"
)

func truckArriveValue() map[string]any {
	return map[string]any{
		"truckWeight": 1000.0,
	}
}

func registerLotValue() map[string]any {
	return map[string]any{
		"client":           "CO",
		"wasteCode":        "160214",
		"numeroFir":        "FIR-1",
		"palletCount":      2.0,
		"truckWeightAfter": 700.0,
		"notes":            "",
	}
}

func adminOp() OperatorIdentity {
	return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
}

func commitTruckArrive(t *testing.T, c *TaskCommitter) string {
	t.Helper()
	return plantCommitArrive(t, c)
}

func commitLotRegister(t *testing.T, c *TaskCommitter, truckID string, value map[string]any) CommitResult {
	t.Helper()
	return plantCommitRegister(t, c, truckID, value)
}

func firstLoadID(domain Domain) string {
	for id := range domain.Kind("lot") {
		return id
	}
	return ""
}

// commitIntakeThroughLotsClosed: arrive → one lot register (weightCheck unlocked; no group Close).
func commitIntakeThroughLotsClosed(t *testing.T, c *TaskCommitter) (truckID, loadID string, domain Domain) {
	t.Helper()
	truckID = commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID = firstLoadID(result.Domain)
	if loadID == "" {
		t.Fatal("expected minted lot")
	}
	stored, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	return truckID, loadID, stored
}

func commitStartLot(t *testing.T, c *TaskCommitter, lotID string) CommitResult {
	t.Helper()
	return plantCommitStartLot(t, c, lotID)
}

func commitWeigh(t *testing.T, c *TaskCommitter, lotID, palletID string, weight float64) CommitResult {
	t.Helper()
	return plantCommitWeigh(t, c, lotID, palletID, weight)
}

func commitMarkProgress(t *testing.T, c *TaskCommitter, lotID string, progress map[string]any) CommitResult {
	t.Helper()
	return plantCommitMarkProgress(t, c, lotID, progress)
}

func commitCloseLot(t *testing.T, c *TaskCommitter, lotID string) CommitResult {
	t.Helper()
	return plantCommitCloseLot(t, c, lotID)
}

// seedLotReadyAfterWeigh opens one lot and weighs both pallets.
// Both return values are the lot id (Job instance id = lot id).
func seedLotReadyAfterWeigh(t *testing.T, c *TaskCommitter) (loadID, lotInstanceID string) {
	t.Helper()
	loadID = plantSeedOpenLotReadyForContainer(t, c)
	return loadID, loadID
}

func completeIntakeThroughPalletWeight(t *testing.T, c *TaskCommitter, _ string) Domain {
	t.Helper()
	_, _ = seedLotReadyAfterWeigh(t, c)
	stored, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func intakeTasks(t *testing.T) Job {
	t.Helper()
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Jobs.Values["intake"].(Job)
}

func lotProcessingJob(t *testing.T) Job {
	t.Helper()
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	job, ok := cfg.Jobs.Values["lot_processing"].(Job)
	if !ok {
		t.Fatal("missing lot_processing Job")
	}
	return job
}

// TestPlantPathLotsAndLotProcessingWeigh locks the feragame plant fixture: many-submit
// register (FakeHttp labels) → weightCheck once, then lot_processing startLot →
// peer weigh/markProgress (Domain weigh-before-progress gate) → closeLot terminal status.
func TestPlantPathLotsAndLotProcessingWeigh(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = NewFakeHttpPort()
	op := adminOp()

	truckID := commitTruckArrive(t, c)
	arriveWL, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	intake, ok := worklistJobByID(arriveWL.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, truckID)
	if !ok || inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register after arrive = %#v", inst)
	}

	lot1 := registerLotValue()
	first := commitLotRegister(t, c, truckID, lot1)
	if len(first.Links) == 0 {
		t.Fatalf("lot register must return label/download links via FakeHttp: %#v", first)
	}
	load1 := firstLoadID(first.Domain)
	if load1 == "" {
		t.Fatal("expected first load")
	}
	intakeAfter, ok := worklistJobByID(first.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake on Commit worklist")
	}
	instAfter, ok := worklistInstance(intakeAfter, truckID)
	if !ok || instAfter.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register after first Commit = %#v", instAfter)
	}

	lot2 := registerLotValue()
	lot2["client"] = "DA"
	lot2["numeroFir"] = "FIR-2"
	lot2["truckWeightAfter"] = 400.0
	second := commitLotRegister(t, c, truckID, lot2)
	if len(second.Links) == 0 {
		t.Fatal("second lot must also return links")
	}
	if len(second.Domain.Kind("lot")) != 2 {
		t.Fatalf("lots = %#v", second.Domain.Kind("lot"))
	}

	mid, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	intake, _ = worklistJobByID(mid.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	if inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register still many = %q", inst.Tasks["register"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck after ≥1 lot = %q, want actionable", inst.Tasks["weightCheck"])
	}

	wc, err := c.Commit(t.Context(), FeragameScope(), op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value:    map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, has := wc.Domain.Kind("truck")[truckID]["checkedLotSum"]; has {
		t.Fatalf("confirm-only weightCheck must not patch checkedLotSum: %#v", wc.Domain.Kind("truck")[truckID])
	}
	afterWC, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	intake, _ = worklistJobByID(afterWC.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	if inst.Tasks["weightCheck"] != TaskStatusDone {
		t.Fatalf("weightCheck after Commit = %q", inst.Tasks["weightCheck"])
	}
	if inst.Tasks["register"] != TaskStatusUnavailable {
		t.Fatalf("register after weightCheck = %q, want unavailable", inst.Tasks["register"])
	}

	lpWL, err := loader.LoadForStreamInstance(t.Context(), FeragameScope(), op, truckID)
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(lpWL.Worklist, "lot_processing")
	if !ok || !lp.CanStart {
		t.Fatalf("lot_processing canStart after intake = %#v", lp)
	}

	_ = commitStartLot(t, c, load1)
	afterStart, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	lp, _ = worklistJobByID(afterStart.Worklist, "lot_processing")
	lotInst, ok := worklistInstance(lp, load1)
	if !ok {
		t.Fatal("missing lot_processing instance")
	}
	if lotInst.Tasks["startLot"] != TaskStatusDone {
		t.Fatalf("startLot after Commit = %q, want done", lotInst.Tasks["startLot"])
	}
	if lotInst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh after startLot = %q, want actionable", lotInst.Tasks["weigh"])
	}
	if lotInst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress after startLot (before weigh) = %q, want unavailable", lotInst.Tasks["markProgress"])
	}
	if lotInst.Tasks["containerReg"] != TaskStatusActionable {
		t.Fatalf("containerReg after startLot = %q, want actionable", lotInst.Tasks["containerReg"])
	}
	if lotInst.Tasks["closeLot"] != TaskStatusUnavailable {
		t.Fatalf("closeLot before weigh+progress = %q, want unavailable", lotInst.Tasks["closeLot"])
	}
	if _, has := lotInst.Tasks["openTurno"]; has {
		t.Fatalf("openTurno must not project: %#v", lotInst.Tasks)
	}
	if _, has := lotInst.Tasks["closeTurno"]; has {
		t.Fatalf("closeTurno must not project: %#v", lotInst.Tasks)
	}

	_ = commitWeigh(t, c, load1, load1+".1", 11)
	weighed := commitWeigh(t, c, load1, load1+".2", 22)
	afterWeigh, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	lp, _ = worklistJobByID(afterWeigh.Worklist, "lot_processing")
	lotInst, _ = worklistInstance(lp, load1)
	if lotInst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress after weigh = %q, want actionable", lotInst.Tasks["markProgress"])
	}
	last := weighed.Domain.Submissions[len(weighed.Domain.Submissions)-1]
	if last.TaskID != "weigh" {
		t.Fatalf("weigh submission = %#v", last)
	}
	if weighed.Domain.Kind("lot")[load1]["status"] != "open" {
		t.Fatalf("lot status after weigh = %#v, want open", weighed.Domain.Kind("lot")[load1])
	}

	_ = commitMarkProgress(t, c, load1, map[string]any{
		load1 + ".1": "done",
		load1 + ".2": "done",
	})
	closed := commitCloseLot(t, c, load1)
	if closed.Domain.Kind("lot")[load1]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot status = %#v", closed.Domain.Kind("lot")[load1])
	}
	afterClose, err := loader.LoadForStreamInstance(t.Context(), FeragameScope(), op, truckID)
	if err != nil {
		t.Fatal(err)
	}
	lp, _ = worklistJobByID(afterClose.Worklist, "lot_processing")
	if !lp.CanStart {
		t.Fatal("lot_processing canStart after closeLot (second lot remains) want true")
	}
}
