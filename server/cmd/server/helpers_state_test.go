package main

import "testing"

func TestComputeAvailability(t *testing.T) {
	def := testRuntimeConfig().Workflow

	cases := []struct {
		name      string
		process   *Process
		available string
	}{
		{name: "nil process", process: nil, available: "1.1"},
		{name: "empty progress", process: processWithDone(), available: "1.1"},
		{name: "partial progress", process: processWithDone("1.1", "1.2"), available: "1.3"},
		{name: "fully done", process: processWithDone("1.1", "1.2", "1.3", "2.1", "2.2", "3.1", "3.2"), available: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			availability := computeAvailability(def, tc.process)
			for _, sub := range orderedSubsteps(def) {
				want := sub.SubstepID == tc.available
				if got := availability[sub.SubstepID]; got != want {
					t.Fatalf("availability[%s]=%t, want %t", sub.SubstepID, got, want)
				}
			}
		})
	}
}

func TestSequenceAndDoneHelpers(t *testing.T) {
	def := testRuntimeConfig().Workflow

	if !isSequenceOK(def, processWithDone(), "1.1") {
		t.Fatal("expected first substep to be sequence-OK")
	}
	if isSequenceOK(def, processWithDone(), "2.1") {
		t.Fatal("expected 2.1 to be blocked when previous steps are pending")
	}
	if !isSequenceOK(def, processWithDone("1.1", "1.2", "1.3"), "2.1") {
		t.Fatal("expected 2.1 to be sequence-OK after 1.x done")
	}
	if isSequenceOK(def, nil, "2.1") {
		t.Fatal("expected nil process to fail sequence check after first substep")
	}

	if isProcessDone(def, processWithDone("1.1")) {
		t.Fatal("expected partially done process to be incomplete")
	}
	if !isProcessDone(def, processWithDone("1.1", "1.2", "1.3", "2.1", "2.2", "3.1", "3.2")) {
		t.Fatal("expected all done process to be complete")
	}
}

func TestNextAvailableHelpers(t *testing.T) {
	def := testRuntimeConfig().Workflow

	if _, ok := nextAvailableSubstep(def, nil); ok {
		t.Fatal("expected no next substep for nil process")
	}
	if _, ok := nextAvailableSubstepForRole(def, nil, "dep1"); ok {
		t.Fatal("expected no next role substep for nil process")
	}

	sub, ok := nextAvailableSubstep(def, processWithDone())
	if !ok || sub.SubstepID != "1.1" {
		t.Fatalf("expected first available substep 1.1, got %#v (ok=%t)", sub, ok)
	}

	roleSub, ok := nextAvailableSubstepForRole(def, processWithDone("1.1"), "dep1")
	if !ok || roleSub.SubstepID != "1.2" {
		t.Fatalf("expected dep1 next available substep 1.2, got %#v (ok=%t)", roleSub, ok)
	}

	roleSub, ok = nextAvailableSubstepForRole(def, processWithDone("1.1", "1.2"), "dep1")
	if !ok || roleSub.SubstepID != "1.3" {
		t.Fatalf("expected dep1 next available substep 1.3, got %#v (ok=%t)", roleSub, ok)
	}

	if _, ok := nextAvailableSubstepForRole(def, processWithDone("1.1", "1.2", "1.3"), "dep1"); ok {
		t.Fatal("expected no dep1 substep available after dep1 work is done")
	}
}

func TestStepTitleHelpers(t *testing.T) {
	def := testRuntimeConfig().Workflow

	title, ok := stepTitleForSubstep(def, "2.1")
	if !ok || title != "Step 2" {
		t.Fatalf("expected step title Step 2 for 2.1, got %q (ok=%t)", title, ok)
	}
	if _, ok := stepTitleForSubstep(def, "9.9"); ok {
		t.Fatal("expected missing substep to return no title")
	}

	process := processWithDone("1.1", "1.2", "1.3")
	sub, step, ok := nextAvailableSubstepWithStep(def, process)
	if !ok || sub.SubstepID != "2.1" || step.Title != "Step 2" {
		t.Fatalf("expected next substep 2.1 in Step 2, got sub=%#v step=%#v ok=%t", sub, step, ok)
	}

	roleSub, roleStep, ok := nextAvailableSubstepForRoleWithStep(def, process, "dep2")
	if !ok || roleSub.SubstepID != "2.1" || roleStep.Title != "Step 2" {
		t.Fatalf("expected dep2 next substep 2.1 in Step 2, got sub=%#v step=%#v ok=%t", roleSub, roleStep, ok)
	}

	doneProcess := processWithDone("1.1", "1.2", "1.3", "2.1", "2.2", "3.1", "3.2")
	if _, _, ok := nextAvailableSubstepWithStep(def, doneProcess); ok {
		t.Fatal("expected no next substep for done process")
	}

	lastTitle, ok := lastStepTitle(def)
	if !ok || lastTitle != "Step 3" {
		t.Fatalf("expected last step title Step 3, got %q (ok=%t)", lastTitle, ok)
	}
}

func processWithDone(doneSubsteps ...string) *Process {
	progress := map[string]ProcessStep{}
	for _, id := range doneSubsteps {
		progress[id] = ProcessStep{State: "done"}
	}
	return &Process{Progress: progress}
}
