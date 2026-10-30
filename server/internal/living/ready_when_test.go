package living

import (
	"testing"
	"time"
)

func TestEvaluateReadyWhenOmitIsTrue(t *testing.T) {
	ok, err := evaluateReadyWhen(EmptyDomain(), nil, "")
	if err != nil || !ok {
		t.Fatalf("omit → true, got ok=%v err=%v", ok, err)
	}
	ok, err = evaluateReadyWhen(EmptyDomain(), "", "")
	if err != nil || !ok {
		t.Fatalf("blank → true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluateReadyWhenLiteralAndDomainExpr(t *testing.T) {
	ok, err := evaluateReadyWhen(EmptyDomain(), "false", "")
	if err != nil || ok {
		t.Fatalf("false → unavailable, got ok=%v err=%v", ok, err)
	}
	ok, err = evaluateReadyWhen(EmptyDomain(), "true", "")
	if err != nil || !ok {
		t.Fatalf("true → available, got ok=%v err=%v", ok, err)
	}

	empty := EmptyDomain()
	ok, err = evaluateReadyWhen(empty, "$count($domain.load.*) >= 1", "")
	if err != nil || ok {
		t.Fatalf("no loads → false, got ok=%v err=%v", ok, err)
	}
	withLoad := empty
	withLoad.Load = EntityMap{"L1": {"client": "NO"}}
	ok, err = evaluateReadyWhen(withLoad, "$count($domain.load.*) >= 1", "")
	if err != nil || !ok {
		t.Fatalf("one load → true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluateReadyWhenRejectsNonString(t *testing.T) {
	_, err := evaluateReadyWhen(EmptyDomain(), 1, "")
	if err == nil {
		t.Fatal("expected invalid-readyWhen")
	}
}

func committerWithReadyWhen(t *testing.T, taskID, readyWhen string, auth Authorizer) (*Committer, *MemoryDomainStore) {
	t.Helper()
	cfg, err := LoadTruckLoopFixture()
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

func TestCommitCodiceFirDeniedViaCerbosWhenReadyWhenFalse(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains := committerWithReadyWhen(t, "codiceFir", "false", NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
		Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-BLOCKED"},
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
	if _, has := after.Load[instanceID]["codiceFir"]; has {
		t.Fatal("codiceFir must not appear when readyWhen denies")
	}
}

func TestCommitCodiceFirAllowedViaCerbosWhenReadyWhenTrue(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	// Domain expr: open only after register mint (one load).
	c, _ := committerWithReadyWhen(t, "codiceFir", "$count($domain.load.*) >= 1", NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
		Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-READY"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Load[instanceID]["codiceFir"] != "FIR-READY" {
		t.Fatalf("domain = %#v", result.Domain.Load[instanceID])
	}
}
