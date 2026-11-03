package living

import (
	"testing"
	"time"
)

// Changing an authored transform must change Commit without a new apply* function.
func TestCommitRespectsTransformChangeInYAML(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	arrive, ok := TaskByID(intake.Tasks, "truckArrive")
	if !ok {
		t.Fatal("missing truckArrive")
	}
	onSubmit := arrive["onSubmit"].(map[string]any)
	transform := onSubmit["transform"].([]any)
	op := transform[0].(map[string]any)
	set := op["set"].(map[string]any)
	set["plantNote"] = "$.turno"

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
	opID := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", opID, CommitRequest{
		TaskPath: "intake/truckArrive", Value: truckArriveValue(),
		Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := result.JobInstanceID
	if result.Domain.Kind("truck")[truckID]["turno"] != "1st" {
		t.Fatalf("truck = %#v", result.Domain.Kind("truck")[truckID])
	}
	if result.Domain.Kind("truck")[truckID]["plantNote"] != "1st" {
		t.Fatalf("transform YAML change should stamp plantNote without a new apply* — truck=%#v", result.Domain.Kind("truck")[truckID])
	}
}

func TestCommitOpenTurnoLabelFromConfigExpr(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"line"},
	}, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantID := "2026-10-19-1st"
	var label string
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "line" && inst.ID == wantID {
			label = inst.Label
		}
	}
	if label != "Turno: 2026-10-19 1°" {
		t.Fatalf("label from config expr = %q", label)
	}
}
