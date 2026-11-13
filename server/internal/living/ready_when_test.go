package living

import (
	"testing"
	"time"
)

func TestEvaluateReadyWhenOmitIsTrue(t *testing.T) {
	ok, err := evaluateReadyWhen(EmptyDomain(), nil, "", nil)
	if err != nil || !ok {
		t.Fatalf("omit → true, got ok=%v err=%v", ok, err)
	}
	ok, err = evaluateReadyWhen(EmptyDomain(), "", "", nil)
	if err != nil || !ok {
		t.Fatalf("blank → true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluateReadyWhenLiteralAndDomainExpr(t *testing.T) {
	ok, err := evaluateReadyWhen(EmptyDomain(), "false", "", nil)
	if err != nil || ok {
		t.Fatalf("false → unavailable, got ok=%v err=%v", ok, err)
	}
	ok, err = evaluateReadyWhen(EmptyDomain(), "true", "", nil)
	if err != nil || !ok {
		t.Fatalf("true → available, got ok=%v err=%v", ok, err)
	}

	empty := EmptyDomain()
	ok, err = evaluateReadyWhen(empty, "$count($domain.lot.*) >= 1", "", nil)
	if err != nil || ok {
		t.Fatalf("no loads → false, got ok=%v err=%v", ok, err)
	}
	withLoad := empty
	putKind(&withLoad, "lot", EntityMap{"L1": {"client": "NO"}})
	ok, err = evaluateReadyWhen(withLoad, "$count($domain.lot.*) >= 1", "", nil)
	if err != nil || !ok {
		t.Fatalf("one load → true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluateReadyWhenRejectsNonString(t *testing.T) {
	_, err := evaluateReadyWhen(EmptyDomain(), 1, "", nil)
	if err == nil {
		t.Fatal("expected invalid-readyWhen")
	}
}

func committerWithReadyWhen(t *testing.T, taskID, readyWhen string, auth Authorizer) (*Committer, *MemoryDomainStore) {
	t.Helper()
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	task, ok := TaskByID(intake.Tasks, taskID)
	if !ok {
		t.Fatalf("missing task %q", taskID)
	}
	task["readyWhen"] = readyWhen

	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg.Version = DefaultFixtureVersion
	if err := configs.Put(t.Context(), "feragame", cfg.Version, cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "feragame", cfg.Version); err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		auth = AllowAllAuthorizer{}
	}
	return &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: auth,
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}, domains
}

func TestCommitWeightCheckDeniedViaCerbosWhenReadyWhenFalse(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains := committerWithReadyWhen(t, "weightCheck", "false", NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("readyWhen false should deny via Cerbos: %v", err)
	}
	after, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Submissions) != len(before.Submissions) {
		t.Fatalf("domain mutated on readyWhen deny: before=%d after=%d", len(before.Submissions), len(after.Submissions))
	}
	if _, has := after.Kind("truck")[truckID]["checkedTruckWeight"]; has {
		t.Fatal("checkedTruckWeight must not appear when readyWhen denies")
	}
}

func TestCommitWeightCheckAllowedViaCerbosWhenReadyWhenTrue(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _ := committerWithReadyWhen(t, "weightCheck", "$count($domain.lot.*) >= 1", NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("truck")[truckID]["checkedTruckWeight"] != 1000.0 {
		t.Fatalf("domain = %#v", result.Domain.Kind("truck")[truckID])
	}
}
