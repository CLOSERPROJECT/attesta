package org

import (
	"testing"
)

func TestFeragameProjectWorklistTwoTrucksIsolateLots(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := FixtureAllRolesOp()

	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"truck": {
			"T-A": {"truckWeight": 1000.0},
			"T-B": {"truckWeight": 1100.0},
		},
		"lot": {
			"L-A1": {"truckId": "T-A", "status": "registered", "client": "A"},
			"L-A2": {"truckId": "T-A", "status": "open", "client": "A"},
			"L-B1": {"truckId": "T-B", "status": "registered", "client": "B"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "intake", ID: "T-B", Label: "T-B"},
		{JobID: "lot_processing", ID: "L-A1", Label: "Lotto: L-A1"},
		{JobID: "lot_processing", ID: "L-A2", Label: "Lotto: L-A2"},
		{JobID: "lot_processing", ID: "L-B1", Label: "Lotto: L-B1"},
	}

	wlA := ProjectWorklist(cfg, domain, op, "T-A")
	intakeA := worklistMustJob(t, wlA, "intake")
	if len(intakeA.Instances) != 1 || intakeA.Instances[0].ID != "T-A" {
		t.Fatalf("truck A intake = %#v", intakeA.Instances)
	}
	lpA := worklistMustJob(t, wlA, "lot_processing")
	gotLots := make([]string, 0, len(lpA.Instances))
	for _, inst := range lpA.Instances {
		gotLots = append(gotLots, inst.ID)
	}
	if len(gotLots) != 2 || gotLots[0] != "L-A1" || gotLots[1] != "L-A2" {
		// order follows Domain JobInstances order
		want := map[string]bool{"L-A1": true, "L-A2": true}
		if len(gotLots) != 2 || !want[gotLots[0]] || !want[gotLots[1]] || gotLots[0] == gotLots[1] {
			t.Fatalf("truck A lots = %#v, want L-A1 and L-A2 only", gotLots)
		}
	}
	for _, inst := range lpA.Instances {
		if inst.ID == "L-B1" {
			t.Fatal("truck B lot leaked into truck A Worklist")
		}
	}

	wlB := ProjectWorklist(cfg, domain, op, "T-B")
	lpB := worklistMustJob(t, wlB, "lot_processing")
	if len(lpB.Instances) != 1 || lpB.Instances[0].ID != "L-B1" {
		t.Fatalf("truck B lots = %#v, want only L-B1", lpB.Instances)
	}
	intakeB := worklistMustJob(t, wlB, "intake")
	if len(intakeB.Instances) != 1 || intakeB.Instances[0].ID != "T-B" {
		t.Fatalf("truck B intake = %#v", intakeB.Instances)
	}
}
