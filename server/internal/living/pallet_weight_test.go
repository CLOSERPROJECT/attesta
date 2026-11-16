package living

import (
	"testing"
)

func TestCommitWeighPatchesOnePalletLeavesOtherUnweighted(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)

	result := commitWeigh(t, c, loadID, loadID+".1", 11)
	if result.Domain.Kind("pallet")[loadID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if _, has := result.Domain.Kind("pallet")[loadID+".2"]["weight"]; has {
		t.Fatalf("pallet .2 must stay unweighted; got %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "weigh" || last.GroupID != "" {
		t.Fatalf("weigh submission = %#v", last)
	}
	if last.Facts["palletId"] != loadID+".1" {
		t.Fatalf("weigh facts = %#v, want palletId %s.1", last.Facts, loadID)
	}
	if _, has := last.Facts["averagePanelWeight"]; !has {
		t.Fatal("averagePanelWeight must stay Submission fact")
	}
}

func TestMarkProgressUnavailableUntilWeigh(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineOp := OperatorIdentity{UserID: "line", Org: "feragame", Roles: []string{"line"}}
	cfg := mustActiveConfig(t, c)

	_ = commitStartLot(t, c, loadID)
	started := mustDomain(t, c)
	if started.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("lot status = %#v", started.Kind("lot")[loadID])
	}
	startWL := ProjectWorklist(cfg, started, lineOp)
	startLP, _ := worklistJobByID(startWL, "lot_processing")
	startInst, ok := worklistInstance(startLP, loadID)
	if !ok || startInst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("open lot without weigh: markProgress = %#v, want unavailable", startInst)
	}
	if startInst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh after startLot (line) = %#v", startInst)
	}
	logWL := ProjectWorklist(cfg, started, OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}})
	logLP, _ := worklistJobByID(logWL, "lot_processing")
	logInst, ok := worklistInstance(logLP, loadID)
	if !ok || logInst.Tasks["weigh"] != TaskStatusAvailable {
		t.Fatalf("weigh after startLot (logistic) = %#v, want available", logInst)
	}

	_ = commitWeigh(t, c, loadID, loadID+".1", 11)
	afterWeigh := ProjectWorklist(cfg, mustDomain(t, c), lineOp)
	weighLP, _ := worklistJobByID(afterWeigh, "lot_processing")
	weighInst, ok := worklistInstance(weighLP, loadID)
	if !ok || weighInst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("after weigh: markProgress = %#v, want actionable", weighInst)
	}
}

func mustDomain(t *testing.T, c *Committer) Domain {
	t.Helper()
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	return domain
}
