package living

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

func openTurnoValue() map[string]any {
	return map[string]any{
		"date":  "2026-10-19",
		"turno": "1st",
	}
}

func adminOp() OperatorIdentity {
	return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration", "logistic", "line"}}
}

// Idempotent when that day Job instance already exists.
func commitOpenWorkingDay(t *testing.T, c *Committer, date, mode string) (string, error) {
	t.Helper()
	if date == "" {
		date = "2026-10-19"
	}
	if mode == "" {
		mode = "multi"
	}
	if domain, err := c.Domains.Get(t.Context(), "feragame"); err == nil {
		if day, ok := domain.Kind("workingDay")[date]; ok {
			if day["status"] == "open" {
				return date, nil
			}
		}
		for _, inst := range domain.JobInstances {
			if inst.JobID == "workingDay" && inst.ID == date {
				return date, nil
			}
		}
	}
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath: "workingDay/open",
		Value:    map[string]any{"date": date, "mode": mode},
		Mint:     startingNewMint(),
	})
	if err != nil {
		return "", err
	}
	return result.JobInstanceID, nil
}

func ensureOpenWorkingDay(t *testing.T, c *Committer) string {
	t.Helper()
	dayID, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi")
	if err != nil {
		t.Fatal(err)
	}
	return dayID
}

func commitTruckArrive(t *testing.T, c *Committer) string {
	t.Helper()
	ensureOpenWorkingDay(t, c)
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
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         value,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func firstLoadID(domain Domain) string {
	for id := range domain.Kind("lot") {
		return id
	}
	return ""
}

// commitIntakeThroughLotsClosed: arrive → one lot register (weightCheck unlocked; no group Close).
func commitIntakeThroughLotsClosed(t *testing.T, c *Committer) (truckID, loadID string, domain Domain) {
	t.Helper()
	truckID = commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID = firstLoadID(result.Domain)
	if loadID == "" {
		t.Fatal("expected minted lot")
	}
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	return truckID, loadID, stored
}

func commitOpenTurnoWith(t *testing.T, c *Committer, value map[string]any) string {
	t.Helper()
	date, _ := value["date"].(string)
	if date == "" {
		date = "2026-10-19"
	}
	ensureOpenWorkingDayForDate(t, c, date)
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

func ensureOpenWorkingDayForDate(t *testing.T, c *Committer, date string) {
	t.Helper()
	if _, err := commitOpenWorkingDay(t, c, date, "multi"); err != nil {
		t.Fatal(err)
	}
}

func commitOpenTurno(t *testing.T, c *Committer) string {
	t.Helper()
	return commitOpenTurnoWith(t, c, openTurnoValue())
}

func commitCloseTurno(t *testing.T, c *Committer, lineID string) {
	t.Helper()
	if _, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/closeTurno",
		JobInstanceID: lineID,
		Value:         map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
}

func commitStartLot(t *testing.T, c *Committer, lineID, lotID string) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/startLot",
		JobInstanceID: lineID,
		Value:         map[string]any{"lotId": lotID},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func commitWeigh(t *testing.T, c *Committer, lineID, palletID string, weight float64) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/weigh",
		JobInstanceID: lineID,
		Value:         weighValue(palletID, weight),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func commitMarkProgress(t *testing.T, c *Committer, lineID string, progress map[string]any) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/markProgress",
		JobInstanceID: lineID,
		Value:         map[string]any{"progress": progress},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func commitLineLoadMark(t *testing.T, c *Committer, lineID string, progress map[string]any) CommitResult {
	t.Helper()
	return commitMarkProgress(t, c, lineID, progress)
}

func commitCloseLot(t *testing.T, c *Committer, lineID string) CommitResult {
	t.Helper()
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/closeLot",
		JobInstanceID: lineID,
		Value:         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func seedLineReadyForLineLoad(t *testing.T, c *Committer) (loadID, lineID string) {
	t.Helper()
	_, loadID, _ = commitIntakeThroughLotsClosed(t, c)
	lineID = commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	_ = commitWeigh(t, c, lineID, loadID+".2", 22)
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

// TestPlantPathLotsAndLineWeigh locks the feragame plant fixture: many-submit
// register (FakeHttp labels) → weightCheck once, then line openTurno → startLot →
// peer weigh/markProgress (Domain weigh-before-progress gate) → closeLot terminal status.
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
	if !ok || inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register after arrive = %#v", inst)
	}
	if _, has := inst.Groups["lots"]; has {
		t.Fatalf("intake must not project lots group chrome: %#v", inst.Groups)
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

	mid, err := loader.Load(t.Context(), "feragame", op)
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
	afterWC, err := loader.Load(t.Context(), "feragame", op)
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
	if !ok || lineInst.Tasks["startLot"] != TaskStatusActionable {
		t.Fatalf("startLot after openTurno = %#v", lineInst)
	}
	if lineInst.Tasks["weigh"] != TaskStatusUnavailable {
		t.Fatalf("weigh before startLot = %q, want unavailable (readyWhen)", lineInst.Tasks["weigh"])
	}
	if lineInst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress before startLot = %q, want unavailable (readyWhen)", lineInst.Tasks["markProgress"])
	}
	if lineInst.Tasks["closeTurno"] != TaskStatusActionable {
		t.Fatalf("closeTurno after openTurno = %q, want actionable", lineInst.Tasks["closeTurno"])
	}
	if len(lineInst.Groups) != 0 {
		t.Fatalf("line must not project group chrome: %#v", lineInst.Groups)
	}

	_ = commitStartLot(t, c, lineID, load1)
	afterStart, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	line, _ = worklistJobByID(afterStart.Worklist, "line")
	lineInst, _ = worklistInstance(line, lineID)
	if lineInst.Tasks["startLot"] != TaskStatusUnavailable {
		t.Fatalf("startLot while lot open = %q, want unavailable", lineInst.Tasks["startLot"])
	}
	if lineInst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh after startLot = %q, want actionable", lineInst.Tasks["weigh"])
	}
	if lineInst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress after startLot (before weigh) = %q, want unavailable", lineInst.Tasks["markProgress"])
	}
	if lineInst.Tasks["containerReg"] != TaskStatusActionable {
		t.Fatalf("containerReg after startLot = %q, want actionable", lineInst.Tasks["containerReg"])
	}
	if lineInst.Tasks["closeLot"] != TaskStatusUnavailable {
		t.Fatalf("closeLot before weigh+progress = %q, want unavailable", lineInst.Tasks["closeLot"])
	}

	_ = commitWeigh(t, c, lineID, load1+".1", 11)
	weighed := commitWeigh(t, c, lineID, load1+".2", 22)
	afterWeigh, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	line, _ = worklistJobByID(afterWeigh.Worklist, "line")
	lineInst, _ = worklistInstance(line, lineID)
	if lineInst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress after weigh = %q, want actionable", lineInst.Tasks["markProgress"])
	}
	last := weighed.Domain.Submissions[len(weighed.Domain.Submissions)-1]
	if last.TaskID != "weigh" || last.GroupID != "" {
		t.Fatalf("weigh submission = %#v", last)
	}
	if weighed.Domain.Kind("lot")[load1]["status"] != "open" {
		t.Fatalf("lot status after weigh = %#v, want open", weighed.Domain.Kind("lot")[load1])
	}

	_ = commitMarkProgress(t, c, lineID, map[string]any{
		load1 + ".1": "done",
		load1 + ".2": "done",
	})
	closed := commitCloseLot(t, c, lineID)
	if closed.Domain.Kind("lot")[load1]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot status = %#v", closed.Domain.Kind("lot")[load1])
	}
	afterClose, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	line, _ = worklistJobByID(afterClose.Worklist, "line")
	lineInst, _ = worklistInstance(line, lineID)
	if lineInst.Tasks["startLot"] != TaskStatusActionable {
		t.Fatalf("startLot after closeLot (second lot remains) = %q", lineInst.Tasks["startLot"])
	}
}
