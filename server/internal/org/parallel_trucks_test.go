package org

import (
	"strings"
	"testing"
)

// Parallel trucks on the recycling stream: two intake trucks at once; line-busy
// is per truck, so truck B can Apri lotto while truck A has an open lot.
func TestParallelTrucksIntakeAndLineBusyBlockedReason(t *testing.T) {
	c, _, _ := testTaskCommitter(t, readyWhenGateAuthorizer{})
	cfg := mustActiveConfig(t, c)
	admin := adminOp()
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	truckA := plantCommitArrive(t, c)
	regA := plantCommitRegister(t, c, truckA, registerLotValue())
	lotA := firstLoadID(regA.Domain)
	if lotA == "" {
		t.Fatal("expected lot on truck A")
	}

	truckB, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatalf("second truckArrive while truck A still incomplete must allow: %v", err)
	}
	if truckB.JobInstanceID == "" || truckB.JobInstanceID == truckA {
		t.Fatalf("truck B id = %q, want distinct from %q", truckB.JobInstanceID, truckA)
	}

	lotBVal := registerLotValue()
	lotBVal["client"] = "DA"
	lotBVal["numeroFir"] = "FIR-B"
	lotBVal["truckWeightAfter"] = 800.0
	regB := plantCommitRegister(t, c, truckB.JobInstanceID, lotBVal)
	lotB := ""
	for id, lot := range regB.Domain.Kind("lot") {
		if tid, _ := lot["truckId"].(string); tid == truckB.JobInstanceID {
			lotB = id
			break
		}
	}
	if lotB == "" {
		t.Fatal("expected lot on truck B")
	}

	netA, _ := regA.Domain.Kind("lot")[lotA]["netWeight"].(float64)
	plantCommitWeightCheck(t, c, truckA, 1000.0, netA)
	started := plantCommitStartLot(t, c, lotA)
	if started.Domain.Kind("lot")[lotA]["status"] != "open" {
		t.Fatalf("lot A status = %#v, want open", started.Domain.Kind("lot")[lotA])
	}

	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	intakeWL := ProjectWorklist(cfg, domain, admin, truckA)
	intakeA, ok := worklistJobByID(intakeWL, "intake")
	if !ok || len(intakeA.Instances) != 1 || intakeA.Instances[0].ID != truckA {
		t.Fatalf("truck A intake view = %#v", intakeA)
	}
	intakeBView := ProjectWorklist(cfg, domain, admin, truckB.JobInstanceID)
	intakeB, ok := worklistJobByID(intakeBView, "intake")
	if !ok || len(intakeB.Instances) != 1 || intakeB.Instances[0].ID != truckB.JobInstanceID {
		t.Fatalf("truck B intake view = %#v", intakeB)
	}

	tl := ProjectStreamTimeline(cfg, domain, logistic, truckB.JobInstanceID)
	for _, step := range tl.Steps {
		if step.JobID != "lot_processing" {
			continue
		}
		for _, row := range step.Rows {
			if row.ID != lotB {
				continue
			}
			if len(row.Tasks) == 0 || row.Tasks[0].ID != "startLot" {
				t.Fatalf("lot B timeline tasks = %#v", row.Tasks)
			}
			if row.Tasks[0].State == StreamSubstepLocked {
				t.Fatalf("Apri lotto state = %q, want not locked (per-truck line-busy)", row.Tasks[0].State)
			}
			if row.Tasks[0].BlockedReason != "" {
				t.Fatalf("blockedReason = %q, want empty", row.Tasks[0].BlockedReason)
			}
		}
	}

	openedB, err := c.Commit(t.Context(), FeragameScope(), logistic, CommitRequest{
		TaskPath:         "lot_processing/startLot",
		Value:            map[string]any{"lotId": lotB},
		Mint:             startingNewMint(),
		StreamInstanceID: truckB.JobInstanceID,
	})
	if err != nil {
		t.Fatalf("startLot for truck B while A open must allow: %v", err)
	}
	if openedB.Domain.Kind("lot")[lotB]["status"] != "open" {
		t.Fatalf("lot B status = %#v, want open", openedB.Domain.Kind("lot")[lotB])
	}
}

func TestFeragameStartLotScopesLotsReadAndBlockedReason(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	lp := cfg.Jobs.Values["lot_processing"].(Job)
	startLot, ok := TaskByID(lp.Tasks, "startLot")
	if !ok {
		t.Fatal("startLot missing")
	}
	rw, _ := startLot["readyWhen"].(string)
	if !strings.Contains(rw, "truckId = $stream.id") {
		t.Fatalf("startLot.readyWhen = %q, want truckId = $stream.id", rw)
	}
	reads, _ := startLot["reads"].(map[string]any)
	lots, _ := reads["lots"].(map[string]any)
	where, _ := lots["where"].(string)
	if !strings.Contains(where, "truckId = $stream.id") {
		t.Fatalf("startLot.reads.lots.where = %q, want truckId = $stream.id", where)
	}
	br, _ := startLot["blockedReason"].(string)
	if !strings.Contains(br, "Linea occupata") || !strings.Contains(br, "camion") {
		t.Fatalf("startLot.blockedReason = %q, want Linea occupata naming lot and truck", br)
	}
	if !strings.Contains(br, "truckId = $stream.id") {
		t.Fatalf("startLot.blockedReason = %q, want truckId = $stream.id sift", br)
	}
	arrive, ok := TaskByID(cfg.Jobs.Values["intake"].(Job).Tasks, "truckArrive")
	if !ok {
		t.Fatal("truckArrive missing")
	}
	if _, has := arrive["readyWhen"]; has {
		t.Fatalf("truckArrive must drop serialising readyWhen, got %#v", arrive["readyWhen"])
	}
	register, _ := TaskByID(cfg.Jobs.Values["intake"].(Job).Tasks, "register")
	httpOps, _ := register["onSubmit"].(map[string]any)
	rawHTTP, _ := httpOps["http"].([]any)
	foundQR := false
	for _, raw := range rawHTTP {
		op, _ := raw.(map[string]any)
		body, _ := op["body"].(string)
		if strings.Contains(body, "$qrEndpoint") {
			foundQR = true
			break
		}
	}
	if !foundQR {
		t.Fatal("register onSubmit.http label body must keep $qrEndpoint")
	}
}
