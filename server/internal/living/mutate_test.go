package living

import "testing"

func TestMutateMintsAndAppendsSubmission(t *testing.T) {
	got, err := Mutate(EmptyDomain(), MutateRequest{
		UserID:        "ada",
		JobID:         "intake",
		TaskID:        "register",
		JobInstanceID: "load-1",
		Timestamp:     "2026-10-22T12:00:00.000Z",
		Facts:         map[string]any{"palletCount": 2, "truckWeight": 900},
		Ops: []MutateOp{
			MintOp{Entity: EntityLoad, ID: "load-1", Set: map[string]any{"truckWeight": 900, "palletCount": 2}},
			MintOp{Entity: EntityPallet, ID: "load-1.1", Set: map[string]any{"loadId": "load-1"}},
		},
		NewJobInstance: &JobInstance{JobID: "intake", ID: "load-1", Label: "load-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Load["load-1"]["truckWeight"] != 900 {
		t.Fatalf("load = %#v", got.Load)
	}
	if len(got.Submissions) != 1 || got.Submissions[0].TaskID != "register" {
		t.Fatalf("submissions = %#v", got.Submissions)
	}
	if got.Submissions[0].JobID != "intake" || got.Submissions[0].JobInstanceID != "load-1" {
		t.Fatalf("submission keys = %#v", got.Submissions[0])
	}
	if len(got.JobInstances) != 1 || got.JobInstances[0].ID != "load-1" {
		t.Fatalf("jobInstances = %#v", got.JobInstances)
	}
}

func TestMutateDuplicateID(t *testing.T) {
	domain := EmptyDomain()
	domain.Load["load-1"] = map[string]any{"client": "NO"}
	_, err := Mutate(domain, MutateRequest{
		UserID: "ada",
		JobID:  "intake",
		TaskID: "register",
		Facts:  map[string]any{},
		Ops:    []MutateOp{MintOp{Entity: EntityLoad, ID: "load-1", Set: map[string]any{}}},
	})
	var me MutateError
	if err == nil || !asMutateError(err, &me) || me.Kind != "duplicate-id" {
		t.Fatalf("err = %v", err)
	}
}

func asMutateError(err error, dest *MutateError) bool {
	if err == nil {
		return false
	}
	if me, ok := err.(MutateError); ok {
		*dest = me
		return true
	}
	return false
}
