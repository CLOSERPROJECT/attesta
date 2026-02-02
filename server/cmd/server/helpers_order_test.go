package main

import (
	"reflect"
	"testing"
)

func TestWorkflowOrderingHelpers(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "2",
				Order:  2,
				Substep: []WorkflowSub{
					{SubstepID: "2.2", Order: 2},
					{SubstepID: "2.1", Order: 1},
				},
			},
			{
				StepID: "1",
				Order:  1,
				Substep: []WorkflowSub{
					{SubstepID: "1.2", Order: 2},
					{SubstepID: "1.1", Order: 1},
				},
			},
		},
	}

	steps := sortedSteps(def)
	if got, want := []string{steps[0].StepID, steps[1].StepID}, []string{"1", "2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedSteps mismatch: got %v want %v", got, want)
	}

	subs := sortedSubsteps(steps[0])
	if got, want := []string{subs[0].SubstepID, subs[1].SubstepID}, []string{"1.1", "1.2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedSubsteps mismatch: got %v want %v", got, want)
	}

	ordered := orderedSubsteps(def)
	got := []string{ordered[0].SubstepID, ordered[1].SubstepID, ordered[2].SubstepID, ordered[3].SubstepID}
	want := []string{"1.1", "1.2", "2.1", "2.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderedSubsteps mismatch: got %v want %v", got, want)
	}
}

func TestProgressKeyEncodingAndNormalization(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wanted string
	}{
		{name: "dot to underscore", input: "1.1", wanted: "1_1"},
		{name: "multiple dots", input: "a.b.c", wanted: "a_b_c"},
		{name: "already underscore", input: "a_b", wanted: "a_b"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodeProgressKey(tc.input); got != tc.wanted {
				t.Fatalf("encodeProgressKey(%q) = %q, want %q", tc.input, got, tc.wanted)
			}
		})
	}

	normalized := normalizeProgressKeys(map[string]ProcessStep{
		"1_1":   {State: "done"},
		"a_b_c": {State: "pending"},
	})

	if _, ok := normalized["1.1"]; !ok {
		t.Fatalf("expected normalized key 1.1, got %#v", normalized)
	}
	if _, ok := normalized["a.b.c"]; !ok {
		t.Fatalf("expected normalized key a.b.c, got %#v", normalized)
	}

	nilNormalized := normalizeProgressKeys(nil)
	if len(nilNormalized) != 0 {
		t.Fatalf("expected empty map for nil progress, got %#v", nilNormalized)
	}
}
