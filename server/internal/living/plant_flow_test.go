package living

import (
	"testing"
)

func truckArriveValue() map[string]any {
	return map[string]any{
		"date":        "2026-10-19",
		"turno":       "1st",
		"truckWeight": 1000.0,
	}
}

func registerLotValue() map[string]any {
	return map[string]any{
		"client":           "NO",
		"wasteCode":        "191212",
		"numeroFir":        "FIR-1",
		"palletCount":      2.0,
		"truckWeightAfter": 700.0,
		"notes":            "",
	}
}

func openTurnoValue() map[string]any {
	return map[string]any{
		"date":  "2026-10-19",
		"turno": "1st",
	}
}

func adminOp() OperatorIdentity {
	return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
}

func commitTruckArrive(t *testing.T, c *Committer) string {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
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

func commitLotRegister(t *testing.T, c *Committer, truckID string, value map[string]any) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         value,
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func closeLots(t *testing.T, c *Committer, truckID string) {
	t.Helper()
	if _, err := c.CloseGroup(t.Context(), "feragame", adminOp(), CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
	}); err != nil {
		t.Fatal(err)
	}
}

func firstLoadID(domain Domain) string {
	for id := range domain.Kind("load") {
		return id
	}
	return ""
}

// commitIntakeThroughLotsClosed: arrive → one lot register → close lots.
func commitIntakeThroughLotsClosed(t *testing.T, c *Committer) (truckID, loadID string, domain Domain) {
	t.Helper()
	truckID = commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID = firstLoadID(result.Domain)
	if loadID == "" {
		t.Fatal("expected minted load")
	}
	closeLots(t, c, truckID)
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	return truckID, loadID, stored
}

func commitOpenTurnoWith(t *testing.T, c *Committer, value map[string]any) string {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath: "line/openTurno",
		Value:    value,
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.JobInstanceID
}

func commitOpenTurno(t *testing.T, c *Committer) string {
	t.Helper()
	return commitOpenTurnoWith(t, c, openTurnoValue())
}

func commitWeigh(t *testing.T, c *Committer, lineID string, weights map[string]any) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      GroupTaskPath("line", "palletWeight", ProvisionalGroupInstanceID, "weigh"),
		JobInstanceID: lineID,
		Value:         palletWeightValue(weights),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func closePalletWeight(t *testing.T, c *Committer, lineID string) {
	t.Helper()
	if _, err := c.CloseGroup(t.Context(), "feragame", adminOp(), CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "palletWeight",
	}); err != nil {
		t.Fatal(err)
	}
}

// seedLineReadyForLineLoad: intake lots + openTurno + weigh all pallets + close weigh group.
func seedLineReadyForLineLoad(t *testing.T, c *Committer) (loadID, lineID string) {
	t.Helper()
	_, loadID, _ = commitIntakeThroughLotsClosed(t, c)
	lineID = commitOpenTurno(t, c)
	_ = commitWeigh(t, c, lineID, map[string]any{
		loadID + ".1": weighRow(11),
		loadID + ".2": weighRow(22),
	})
	closePalletWeight(t, c, lineID)
	return loadID, lineID
}

func completeIntakeThroughPalletWeight(t *testing.T, c *Committer, _ string) Domain {
	t.Helper()
	_, _ = seedLineReadyForLineLoad(t, c)
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func commitLineShift(t *testing.T, c *Committer, shift string) Domain {
	t.Helper()
	val := openTurnoValue()
	if shift != "" && shift != "1st" {
		val["turno"] = shift
	}
	wantID := val["date"].(string) + "-" + val["turno"].(string)
	if domain, err := c.Domains.Get(t.Context(), "feragame"); err == nil {
		for _, inst := range domain.JobInstances {
			if inst.JobID == "line" && inst.ID == wantID {
				return domain
			}
		}
	}
	_ = commitOpenTurnoWith(t, c, val)
	stored, err := c.Domains.Get(t.Context(), "feragame")
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

func lineJob(t *testing.T) Job {
	t.Helper()
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Jobs.Values["line"].(Job)
}

func palletWeightTask(line Job) TaskConfig {
	v, _ := TaskByID(line.Tasks, "palletWeight")
	return v
}

// TestPlantPathLotsAndLineWeigh locks the feragame plant fixture: several intake
// lots (FakeHttp labels), Close → weightCheck, then line weigh-series Close → lineLoad.
func TestPlantPathLotsAndLineWeigh(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = NewFakeHttpPort()
	op := adminOp()

	truckID := commitTruckArrive(t, c)
	arriveWL, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	intake, ok := worklistJobByID(arriveWL.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, truckID)
	if !ok || inst.Tasks["lots"] != TaskStatusAvailable {
		t.Fatalf("lots after arrive = %#v", inst)
	}
	if g := inst.Groups["lots"]; !g.CanSpawn || g.CanClose {
		t.Fatalf("open empty lots chrome = %#v", g)
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

	lot2 := registerLotValue()
	lot2["client"] = "SU"
	lot2["numeroFir"] = "FIR-2"
	lot2["truckWeightAfter"] = 400.0
	second := commitLotRegister(t, c, truckID, lot2)
	if len(second.Links) == 0 {
		t.Fatal("second lot must also return links")
	}
	if len(second.Domain.GroupInstances) != 2 {
		t.Fatalf("group instances = %#v", second.Domain.GroupInstances)
	}

	mid, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	intake, _ = worklistJobByID(mid.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	if inst.Tasks["weightCheck"] != TaskStatusLocked {
		t.Fatalf("weightCheck before close = %q", inst.Tasks["weightCheck"])
	}
	if g := inst.Groups["lots"]; !g.CanSpawn || !g.CanClose {
		t.Fatalf("lots with children = %#v", g)
	}

	closeLots(t, c, truckID)
	afterClose, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	intake, _ = worklistJobByID(afterClose.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	if inst.Tasks["lots"] != TaskStatusDone {
		t.Fatalf("lots after close = %q", inst.Tasks["lots"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck after close = %q, want actionable", inst.Tasks["weightCheck"])
	}
	if g := inst.Groups["lots"]; g.CanSpawn || g.CanClose {
		t.Fatalf("closed lots chrome = %#v", g)
	}

	wc, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 600.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if wc.Domain.Kind("truck")[truckID]["checkedLotSum"] != 600.0 {
		t.Fatalf("weightCheck patch = %#v", wc.Domain.Kind("truck")[truckID])
	}

	lineID := commitOpenTurno(t, c)
	lineWL, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	line, ok := worklistJobByID(lineWL.Worklist, "line")
	if !ok {
		t.Fatal("missing line")
	}
	lineInst, ok := worklistInstance(line, lineID)
	if !ok || lineInst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight after openTurno = %#v", lineInst)
	}
	if g := lineInst.Groups["palletWeight"]; !g.CanSpawn {
		t.Fatalf("weigh group chrome = %#v", g)
	}

	loads := make([]string, 0, 2)
	for id := range wc.Domain.Kind("load") {
		loads = append(loads, id)
	}
	if len(loads) != 2 {
		t.Fatalf("loads = %#v", loads)
	}
	weights := map[string]any{}
	for _, loadID := range loads {
		weights[loadID+".1"] = weighRow(11)
		weights[loadID+".2"] = weighRow(22)
	}
	weighed := commitWeigh(t, c, lineID, weights)
	last := weighed.Domain.Submissions[len(weighed.Domain.Submissions)-1]
	if last.TaskID != "weigh" || last.GroupID != "palletWeight" {
		t.Fatalf("weigh submission = %#v", last)
	}

	closePalletWeight(t, c, lineID)
	ready, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	line, _ = worklistJobByID(ready.Worklist, "line")
	lineInst, _ = worklistInstance(line, lineID)
	if lineInst.Tasks["palletWeight"] != TaskStatusDone {
		t.Fatalf("palletWeight after close = %q", lineInst.Tasks["palletWeight"])
	}
	if lineInst.Tasks["lineLoad"] != TaskStatusActionable {
		t.Fatalf("lineLoad after weigh close = %q, want actionable", lineInst.Tasks["lineLoad"])
	}
}
