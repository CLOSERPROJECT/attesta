package living

import (
	"encoding/json"
	"testing"
)

func TestCommitWorkingDayOpenNavigatesHome(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	opened, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "workingDay/open",
		Value:    map[string]any{"date": "2026-10-19", "mode": "multi"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Navigation != CommitNextHome {
		t.Fatalf("after workingDay/open Navigation = %q, want home", opened.Navigation)
	}
	if opened.NextTaskFormBundle != nil {
		t.Fatalf("home Commit must omit nextTaskFormBundle, got %s", bundleAddress(opened.NextTaskFormBundle))
	}
	if opened.ContinueNavigation != "" {
		t.Fatalf("home Commit must omit continueNavigation, got %q", opened.ContinueNavigation)
	}
}

func TestCommitLineOpenTurnoNavigatesHome(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	ensureOpenWorkingDay(t, c)

	opened, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Navigation != CommitNextHome {
		t.Fatalf("after line/openTurno Navigation = %q, want home", opened.Navigation)
	}
	if opened.NextTaskFormBundle != nil {
		t.Fatalf("home Commit must omit nextTaskFormBundle, got %s", bundleAddress(opened.NextTaskFormBundle))
	}
}

func TestNextAdvanceTaskRefSkipsAdvanceTargetFalse(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	cfg := mustActiveConfig(t, c)
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	lineID := commitOpenTurno(t, c)

	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	wl := ProjectWorklist(cfg, domain, lineOp)
	lineWL, ok := worklistJobByID(wl, "line")
	if !ok {
		t.Fatal("missing line")
	}
	inst, ok := worklistInstance(lineWL, lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	if inst.Tasks["closeTurno"] != TaskStatusActionable {
		t.Fatalf("closeTurno = %q, want actionable so skip can be observed", inst.Tasks["closeTurno"])
	}
	if _, ok := NextAdvanceTaskRef(cfg, wl, "line", lineID); ok {
		t.Fatal("NextAdvanceTaskRef must skip closeTurno when advanceTarget is false")
	}

	c2, _, _ := testCommitter(t, AllowAllAuthorizer{})
	cfg2 := mustActiveConfig(t, c2)
	wdAdmin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	dayID, err := commitOpenWorkingDay(t, c2, "2026-10-19", "multi")
	if err != nil {
		t.Fatal(err)
	}
	domain2, err := c2.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	wl2 := ProjectWorklist(cfg2, domain2, wdAdmin)
	wdWL, ok := worklistJobByID(wl2, "workingDay")
	if !ok {
		t.Fatal("missing workingDay")
	}
	dayInst, ok := worklistInstance(wdWL, dayID)
	if !ok {
		t.Fatal("missing day instance")
	}
	if dayInst.Tasks["close"] != TaskStatusActionable {
		t.Fatalf("close = %q, want actionable so skip can be observed", dayInst.Tasks["close"])
	}
	if _, ok := NextAdvanceTaskRef(cfg2, wl2, "workingDay", dayID); ok {
		t.Fatal("NextAdvanceTaskRef must skip workingDay/close when advanceTarget is false")
	}
}

func TestCloseGroupWhenOnlyCloseTurnoLeftNavigatesOverview(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	op := FixtureAllRolesOp()
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	_ = commitWeigh(t, c, lineID, loadID+".2", 22)
	if _, err := c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "palletWeight",
	}); err != nil {
		t.Fatal(err)
	}
	_ = commitLineLoadMark(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
	if _, err := c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "lineLoad",
	}); err != nil {
		t.Fatal(err)
	}
	_ = commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 50.0}},
	}))

	closed, err := c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "containerReg",
	})
	if err != nil {
		t.Fatal(err)
	}
	lineWL, ok := worklistJobByID(closed.Worklist, "line")
	if !ok {
		t.Fatal("missing line")
	}
	inst, ok := worklistInstance(lineWL, lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	if inst.Tasks["closeTurno"] != TaskStatusActionable {
		t.Fatalf("closeTurno = %q, want actionable remaining work", inst.Tasks["closeTurno"])
	}
	if closed.Navigation != CommitNextOverview {
		t.Fatalf("after last group Close Navigation = %q, want overview (not closeTurno)", closed.Navigation)
	}
	if closed.NextTaskFormBundle != nil {
		t.Fatalf("must not auto-advance to closeTurno, got %s", bundleAddress(closed.NextTaskFormBundle))
	}
}

func TestCommitLinkStayContinueNavigationWire(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)

	lot, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         registerLotValue(),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if lot.Navigation != CommitNextStay {
		t.Fatalf("link register Navigation = %q, want stay", lot.Navigation)
	}
	if lot.ContinueNavigation != CommitNextOverview {
		t.Fatalf("ContinueNavigation = %q, want overview while lots open", lot.ContinueNavigation)
	}

	raw, err := json.Marshal(lot)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, has := wire["next"]; has {
		t.Fatalf("Commit JSON must not ship legacy next: %s", string(raw))
	}
	if wire["navigation"] != string(CommitNextStay) {
		t.Fatalf("JSON navigation = %#v", wire["navigation"])
	}
	if wire["continueNavigation"] != string(CommitNextOverview) {
		t.Fatalf("JSON continueNavigation = %#v", wire["continueNavigation"])
	}
}
