package org

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
	set["plantNote"] = `"from-yaml"`

	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg.Version = DefaultFixtureVersion
	if err := configs.Put(t.Context(), "feragame", cfg.Version, cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "feragame", cfg.Version); err != nil {
		t.Fatal(err)
	}
	c := &TaskCommitter{
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
	if _, has := result.Domain.Kind("truck")[truckID]["workingDayId"]; has {
		t.Fatalf("truck must not stamp workingDayId: %#v", result.Domain.Kind("truck")[truckID])
	}
	if result.Domain.Kind("truck")[truckID]["plantNote"] != "from-yaml" {
		t.Fatalf("transform YAML change should stamp plantNote without a new apply* — truck=%#v", result.Domain.Kind("truck")[truckID])
	}
}

func TestCommitStartLotLabelFromConfigExpr(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID := plantIntakeOneLot(t, c)
	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"logistic"},
	}, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantLabel := "Lotto: " + loadID
	var label string
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "lot_processing" && inst.ID == loadID {
			label = inst.Label
		}
	}
	if label != wantLabel {
		t.Fatalf("label from config expr = %q, want %q", label, wantLabel)
	}
}
