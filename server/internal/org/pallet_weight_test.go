package org

import (
	"testing"
)

// Timeline after stay reload uses BuildOrgTimelineView: weigh (submit:many) stays
// org-actionable while ≥1 unweighted pallet remains; after the last weigh it maps
// readyWhen-false → unavailable → locked (never done — ADR-0008).
func TestTimelineWeighStaysAvailableUntilLastUnweightedPallet(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID := plantIntakeOneLot(t, c)
	_ = plantCommitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	lineOp := OperatorIdentity{UserID: "line", Org: "feragame", Roles: []string{"line"}}
	focus := mustTimelineInstance(t, truckID)

	assertWeigh := func(when string, wantStatus string, wantActionable bool) {
		t.Helper()
		view := BuildOrgTimelineView(cfg, mustDomain(t, c), lineOp, "recycling", focus)
		weigh := lotProcessingWeighSubstep(t, view, loadID)
		if weigh.Status != wantStatus {
			t.Fatalf("weigh status %s = %q, want %q (submitMany=%v orgActionable=%v)",
				when, weigh.Status, wantStatus, weigh.SubmitMany, weigh.OrgActionable)
		}
		if weigh.OrgActionable != wantActionable {
			t.Fatalf("weigh orgActionable %s = %v, want %v (status=%q)",
				when, weigh.OrgActionable, wantActionable, weigh.Status)
		}
		if !weigh.SubmitMany {
			t.Fatalf("weigh submitMany %s = false, want true", when)
		}
	}

	assertWeigh("after startLot", string(StreamSubstepAvailable), true)
	_ = plantCommitWeigh(t, c, loadID, loadID+".1", 11)
	assertWeigh("after first of two pallets", string(StreamSubstepAvailable), true)
	_ = plantCommitWeigh(t, c, loadID, loadID+".2", 22)
	assertWeigh("after last unweighted pallet", string(StreamSubstepLocked), false)
}

func lotProcessingWeighSubstep(t *testing.T, view TimelineView, lotID string) TimelineSubstepDTO {
	t.Helper()
	for _, step := range view.Steps {
		for _, row := range step.Rows {
			if row.ID != lotID {
				continue
			}
			for _, sub := range row.Substeps {
				if sub.SubstepID == "lot_processing.weigh" {
					return sub
				}
			}
			t.Fatalf("lot %s row missing weigh: %#v", lotID, row.Substeps)
		}
	}
	t.Fatalf("lot row %s missing in %#v", lotID, view.Steps)
	return TimelineSubstepDTO{}
}

func TestCommitWeighPatchesOnePalletLeavesOtherUnweighted(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
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
	if last.TaskID != "weigh" {
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
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineOp := OperatorIdentity{UserID: "line", Org: "feragame", Roles: []string{"line"}}
	cfg := mustActiveConfig(t, c)

	_ = commitStartLot(t, c, loadID)
	started := mustDomain(t, c)
	if started.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("lot status = %#v", started.Kind("lot")[loadID])
	}
	startWL := projectWorklistUnfiltered(cfg, started, lineOp)
	startLP, _ := worklistJobByID(startWL, "lot_processing")
	startInst, ok := worklistInstance(startLP, loadID)
	if !ok || startInst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("open lot without weigh: markProgress = %#v, want unavailable", startInst)
	}
	if startInst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh after startLot (line) = %#v", startInst)
	}
	logWL := projectWorklistUnfiltered(cfg, started, OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}})
	logLP, _ := worklistJobByID(logWL, "lot_processing")
	logInst, ok := worklistInstance(logLP, loadID)
	if !ok || logInst.Tasks["weigh"] != TaskStatusAvailable {
		t.Fatalf("weigh after startLot (logistic) = %#v, want available", logInst)
	}

	_ = commitWeigh(t, c, loadID, loadID+".1", 11)
	afterWeigh := projectWorklistUnfiltered(cfg, mustDomain(t, c), lineOp)
	weighLP, _ := worklistJobByID(afterWeigh, "lot_processing")
	weighInst, ok := worklistInstance(weighLP, loadID)
	if !ok || weighInst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("after weigh: markProgress = %#v, want actionable", weighInst)
	}
}

func mustDomain(t *testing.T, c *TaskCommitter) Domain {
	t.Helper()
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	return domain
}
