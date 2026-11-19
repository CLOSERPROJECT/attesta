package org

import (
	"context"
	"strings"
	"testing"
)

// readyWhenOnlyAuthorizer proves Domain readyWhen alone (SequenceOk ignored).
type readyWhenOnlyAuthorizer struct{}

func (readyWhenOnlyAuthorizer) AllowCommit(_ context.Context, q AuthorizeCommit) (bool, error) {
	if !q.ReadyWhenOk {
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

func TestIntakeReadyWhenRequiresTruckArriveSubmission(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	register, ok := TaskByID(intake.Tasks, "register")
	if !ok {
		t.Fatal("register missing")
	}
	weightCheck, ok := TaskByID(intake.Tasks, "weightCheck")
	if !ok {
		t.Fatal("weightCheck missing")
	}

	for _, task := range []TaskConfig{register, weightCheck} {
		rw, _ := task["readyWhen"].(string)
		if !strings.Contains(rw, `taskId = "truckArrive"`) {
			t.Fatalf("%s readyWhen = %q, want truckArrive Submission gate", task["id"], rw)
		}
	}
	regRW, _ := register["readyWhen"].(string)
	if !strings.Contains(regRW, `taskId = "weightCheck"`) {
		t.Fatalf("register readyWhen must keep weightCheck clause: %q", regRW)
	}
	wcRW, _ := weightCheck["readyWhen"].(string)
	if !strings.Contains(wcRW, "lot") {
		t.Fatalf("weightCheck readyWhen must keep ≥1 lot clause: %q", wcRW)
	}

	beforeArrive := EmptyDomain()
	beforeArrive.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
	putKind(&beforeArrive, "truck", EntityMap{"T1": {"truckWeight": 1000.0}})
	putKind(&beforeArrive, "lot", EntityMap{"L1": {"truckId": "T1", "netWeight": 100.0}})

	if readyWhenOkForCommit(beforeArrive, taskPayload(register), "T1", cfg.Catalogs) {
		t.Fatal("register readyWhen must fail closed without truckArrive Submission")
	}
	if readyWhenOkForCommit(beforeArrive, taskPayload(weightCheck), "T1", cfg.Catalogs) {
		t.Fatal("weightCheck readyWhen must fail closed without truckArrive Submission (even with lots)")
	}

	afterArrive := beforeArrive
	afterArrive.Submissions = []Submission{sub("intake", "truckArrive", "T1")}
	if !readyWhenOkForCommit(afterArrive, taskPayload(register), "T1", cfg.Catalogs) {
		t.Fatal("register readyWhen must hold after truckArrive (no weightCheck yet)")
	}
	if !readyWhenOkForCommit(afterArrive, taskPayload(weightCheck), "T1", cfg.Catalogs) {
		t.Fatal("weightCheck readyWhen must hold after truckArrive with ≥1 lot")
	}
}

func TestCommitRegisterDeniedByReadyWhenBeforeArrive(t *testing.T) {
	c, domains, _ := testTaskCommitter(t, readyWhenOnlyAuthorizer{})
	admin := adminOp()
	cfg := mustActiveConfig(t, c)

	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
	putKind(&domain, "truck", EntityMap{"T1": {"truckWeight": 1000.0, "lastScaleReading": 1000.0}})
	if err := domains.Put(t.Context(), FeragameScope(), domain); err != nil {
		t.Fatal(err)
	}

	register, _ := TaskByID(cfg.Jobs.Values["intake"].(Job).Tasks, "register")
	if readyWhenOkForCommit(domain, taskPayload(register), "T1", cfg.Catalogs) {
		t.Fatal("precondition: register readyWhen must be false before arrive")
	}

	_, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "T1",
		Value:         registerLotValue(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("register Commit before truckArrive must deny via readyWhen: %v", err)
	}

	_, err = c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath:      "intake/weightCheck",
		JobInstanceID: "T1",
		Value:         map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 100.0},
	})
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("weightCheck Commit before truckArrive must deny via readyWhen: %v", err)
	}
}

func TestLotProcessingChildrenReadyWhenCoverStartLot(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	lp := cfg.Jobs.Values["lot_processing"].(Job)

	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "lot_processing", ID: "L1", Label: "L1"}}
	putKind(&domain, "lot", EntityMap{"L1": map[string]any{}})
	putKind(&domain, "pallet", EntityMap{
		"L1.1": {"lotId": "L1"},
		"L1.2": {"lotId": "L1"},
	})

	for _, id := range []string{"weigh", "markProgress", "containerReg", "closeLot"} {
		task, ok := TaskByID(lp.Tasks, id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		rw, _ := task["readyWhen"].(string)
		if strings.Contains(rw, `taskId = "startLot"`) {
			t.Fatalf("%s readyWhen must use Domain lot status, not startLot Submission: %q", id, rw)
		}
		if !strings.Contains(rw, `status = "open"`) {
			t.Fatalf("%s readyWhen = %q, want open-lot Domain gate", id, rw)
		}
		if readyWhenOkForCommit(domain, taskPayload(task), "L1", cfg.Catalogs) {
			t.Fatalf("%s readyWhen must fail closed before startLot opens the lot", id)
		}
	}

	open := domain
	putKind(&open, "lot", EntityMap{"L1": {"status": "open"}})
	weigh, _ := TaskByID(lp.Tasks, "weigh")
	if !readyWhenOkForCommit(open, taskPayload(weigh), "L1", cfg.Catalogs) {
		t.Fatal("weigh readyWhen must hold once lot is open with unweighed pallets")
	}
}

func TestLotProcessingLockReadyWhenGatesOnFullyProcessedUnlocked(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	job := cfg.Jobs.Values["lot_processing"].(Job)
	lock, ok := TaskByID(job.Tasks, "lock")
	if !ok {
		t.Fatal("lock missing")
	}
	rw, _ := lock["readyWhen"].(string)
	if strings.Contains(rw, "select") || strings.Contains(rw, "submissions") {
		t.Fatalf("lock readyWhen = %q, must not gate on select Submission", rw)
	}
	if !strings.Contains(rw, "fullyProcessed") || !strings.Contains(rw, "locked") {
		t.Fatalf("lock readyWhen must keep Domain eligibility: %q", rw)
	}

	eligible := EmptyDomain()
	eligible.JobInstances = []JobInstance{{JobID: "lot_processing", ID: "L1", Label: "L1"}}
	putKind(&eligible, "lot", EntityMap{"L1": {"status": "fullyProcessed"}})
	if !readyWhenOkForCommit(eligible, taskPayload(lock), "L1", cfg.Catalogs) {
		t.Fatal("lock readyWhen must hold on fullyProcessed unlocked lot")
	}
	locked := eligible
	putKind(&locked, "lot", EntityMap{"L1": {"status": "fullyProcessed", "locked": true}})
	if readyWhenOkForCommit(locked, taskPayload(lock), "L1", cfg.Catalogs) {
		t.Fatal("lock readyWhen must fail when lot is locked")
	}
	open := eligible
	putKind(&open, "lot", EntityMap{"L1": {"status": "open"}})
	if readyWhenOkForCommit(open, taskPayload(lock), "L1", cfg.Catalogs) {
		t.Fatal("lock readyWhen must fail while lot is still open")
	}
}
