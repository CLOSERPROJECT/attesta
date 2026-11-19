package org

import (
	"testing"
)

func commitMarkProgressAllDone(t *testing.T, c *TaskCommitter, lotID, loadID string) {
	t.Helper()
	_ = commitMarkProgress(t, c, lotID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
}

func containerRegValue(loadID string, materials map[string][]any) map[string]any {
	v := map[string]any{"sourceLotId": loadID}
	for k, rows := range materials {
		v[k] = rows
	}
	return v
}

func commitContainerReg(t *testing.T, c *TaskCommitter, lotID string, value map[string]any) CommitResult {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: lotID,
		Value:         value,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func seedSecondSULoad(t *testing.T, c *TaskCommitter) string {
	t.Helper()
	truckID := commitTruckArrive(t, c)
	val := registerLotValue()
	val["client"] = "DA"
	val["palletCount"] = 1.0
	val["truckWeightAfter"] = 650.0
	result := commitLotRegister(t, c, truckID, val)
	for id, facts := range result.Domain.Kind("lot") {
		if facts["client"] == "DA" {
			return id
		}
	}
	return firstLoadID(result.Domain)
}

func TestContainerRegActionableOnOpenLotWithoutMarkProgress(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, lotID := seedLotReadyAfterWeigh(t, c)
	domain := mustDomain(t, c)

	cfg := mustActiveConfig(t, c)
	wl := projectWorklistUnfiltered(cfg, domain, OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}})
	inst, ok := worklistInstance(worklistMustJob(t, wl, "lot_processing"), lotID)
	if !ok {
		t.Fatal("missing lot_processing")
	}
	if inst.Tasks["containerReg"] != TaskStatusActionable {
		t.Fatalf("containerReg = %q, want actionable on open lot", inst.Tasks["containerReg"])
	}
}

func TestCommitContainerRegMintsContainersLeavesLotOpen(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)
	result := commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 80.0}},
	}))
	if result.Domain.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("lot status = %#v, want open", result.Domain.Kind("lot")[loadID])
	}
	if len(result.Domain.Kind("container")) == 0 {
		t.Fatal("expected containers")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "containerReg" || last.JobInstanceID != lotID {
		t.Fatalf("submission = %#v", last)
	}
}

func TestContainerRegManySubmitSameOpenLot(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)
	_ = commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 40.0}},
	}))
	second := commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"plastica": {map[string]any{"weight": 20.0}},
	}))
	var n int
	for _, s := range second.Domain.Submissions {
		if s.JobID == "lot_processing" && s.TaskID == "containerReg" && s.JobInstanceID == lotID {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("containerReg submissions = %d, want 2", n)
	}
	if second.Domain.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("lot must stay open across many containerReg: %#v", second.Domain.Kind("lot")[loadID])
	}
}

func TestContainerRegIncompleteContinuationPrefill(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)
	first := commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 50.0, "incomplete": true}},
	}))
	var contID string
	for id, facts := range first.Domain.Kind("container") {
		if facts["incomplete"] == true {
			contID = id
			break
		}
	}
	if contID == "" {
		t.Fatal("expected incomplete container")
	}
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "lot_processing", "containerReg", lotID)
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := bundle.InitialData["polvereDiVetro"].([]any)
	if len(rows) == 0 {
		t.Fatalf("initialData prefill = %#v", bundle.InitialData)
	}
	row, _ := rows[0].(map[string]any)
	if row["continuesContainerId"] != contID {
		t.Fatalf("continuesContainerId = %#v, want %s", row["continuesContainerId"], contID)
	}
}

func TestContainerRegThenCloseLotPath(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)
	commitMarkProgressAllDone(t, c, lotID, loadID)
	_ = commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 120.0}},
		"plastica":       {map[string]any{"weight": 40.0}},
	}))
	closed := commitCloseLot(t, c, lotID)
	if closed.Domain.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("after closeLot = %#v", closed.Domain.Kind("lot")[loadID])
	}
	var sawMark, sawReg, sawClose bool
	for _, s := range closed.Domain.Submissions {
		if s.JobID != "lot_processing" {
			continue
		}
		switch s.TaskID {
		case "markProgress":
			sawMark = true
		case "containerReg":
			sawReg = true
		case "closeLot":
			sawClose = true
		}
	}
	if !sawMark || !sawReg || !sawClose {
		t.Fatalf("submissions missing mark/reg/close: mark=%v reg=%v close=%v", sawMark, sawReg, sawClose)
	}
}

func loadEnumIDs(t *testing.T, bundle *TaskFormBundle, property string) []string {
	t.Helper()
	props, _ := bundle.Schema["properties"].(map[string]any)
	prop, _ := props[property].(map[string]any)
	raw, ok := prop["enum"]
	if !ok {
		t.Fatalf("%s enum missing: %#v", property, prop)
	}
	switch ids := raw.(type) {
	case []string:
		return ids
	case []any:
		out := make([]string, 0, len(ids))
		for _, v := range ids {
			s, _ := v.(string)
			out = append(out, s)
		}
		return out
	default:
		t.Fatalf("%s enum type = %T", property, raw)
		return nil
	}
}

func TestSecondLotRequiresCloseThenStart(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadA, lotID := seedLotReadyAfterWeigh(t, c)
	loadB := seedSecondSULoad(t, c)

	commitMarkProgressAllDone(t, c, lotID, loadA)
	_ = commitContainerReg(t, c, lotID, containerRegValue(loadA, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 10.0}},
	}))
	_ = commitCloseLot(t, c, lotID)

	started := commitStartLot(t, c, loadB)
	if started.Domain.Kind("lot")[loadB]["status"] != "open" {
		t.Fatalf("second startLot = %#v", started.Domain.Kind("lot")[loadB])
	}
	if started.Domain.Kind("lot")[loadA]["status"] != "fullyProcessed" {
		t.Fatalf("first lot = %#v", started.Domain.Kind("lot")[loadA])
	}
}
