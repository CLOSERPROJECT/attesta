package living

import (
	"testing"
	"time"
)

// Changing an authored transform must change Commit without a new apply* function.
func TestCommitRespectsTransformChangeInYAML(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	taskVal, _ := intake.Tasks.Get("codiceFir")
	task := taskVal.(map[string]any)
	onSubmit := task["onSubmit"].(map[string]any)
	transform := onSubmit["transform"].([]any)
	op := transform[0].(map[string]any)
	set := op["set"].(map[string]any)
	// Patch an extra fact from the form value via YAML-shaped transform only.
	set["plantNote"] = "$.codiceFir"

	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg.Version = DefaultFixtureVersion
	if err := configs.Put(t.Context(), "feragame", cfg.Version, cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "feragame", cfg.Version); err != nil {
		t.Fatal(err)
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	instanceID := "NIR-261019-NO-1"
	opID := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	if _, err := c.Commit(t.Context(), "feragame", opID, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
	}); err != nil {
		t.Fatal(err)
	}
	result, err := c.Commit(t.Context(), "feragame", opID, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-YAML"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Load[instanceID]["codiceFir"] != "FIR-YAML" {
		t.Fatalf("codiceFir = %#v", result.Domain.Load[instanceID])
	}
	if result.Domain.Load[instanceID]["plantNote"] != "FIR-YAML" {
		t.Fatalf("transform YAML change should stamp plantNote without a new apply* — load=%#v", result.Domain.Load[instanceID])
	}
}

func TestCommitShiftLabelFromConfigExpr(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"line"},
	}, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: "1st",
		Value: map[string]any{"shift": "1st"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var label string
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "line" && inst.ID == "1st" {
			label = inst.Label
		}
	}
	if label != "Turno: 1st" {
		t.Fatalf("label from config expr = %q", label)
	}
}
