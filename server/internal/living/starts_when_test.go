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
			"weights": map[string]any{"read": "rows"},
		},
	}
	if IsStartsWhenSatisfied(gate, nil, instances, eligible) {
		t.Fatal("some unsatisfied without Submission")
	}
	if !IsStartsWhenSatisfied(gate, []Submission{sub("intake", "palletWeight", "load-1")}, instances, eligible) {
		t.Fatal("some satisfied with matching Submission")
	}
	if IsStartsWhenSatisfied(gate, []Submission{sub("intake", "palletWeight", "load-1")}, instances, nil) {
		t.Fatal("missing referenced Task must fail closed")
	}
}
