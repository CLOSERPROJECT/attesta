package living

import "testing"

func TestIsStartsWhenSatisfiedSomeRequiresMatchingSubmission(t *testing.T) {
	gate := &StartsWhen{
		Task:       StartsWhenRef{JobID: "intake", TaskID: "palletWeight"},
		Quantifier: "some",
	}
	instances := []JobInstance{{JobID: "intake", ID: "load-1", Label: "load-1"}}
	eligible := map[string]any{
		"dynamicProperties": map[string]any{
			"weights": map[string]any{"read": "leftovers", "completeness": "all"},
		},
	}
	if IsStartsWhenSatisfied(gate, nil, instances, eligible) {
		t.Fatal("some unsatisfied without Submission")
	}
	if !IsStartsWhenSatisfied(gate, []Submission{sub("intake", "palletWeight", "load-1")}, instances, eligible) {
		t.Fatal("some satisfied with matching Submission")
	}

	someTask := map[string]any{
		"dynamicProperties": map[string]any{
			"progress": map[string]any{"read": "leftovers", "completeness": "some"},
		},
	}
	if IsStartsWhenSatisfied(gate, []Submission{sub("intake", "palletWeight", "load-1")}, instances, someTask) {
		t.Fatal("completeness some must never satisfy startsWhen")
	}
}
