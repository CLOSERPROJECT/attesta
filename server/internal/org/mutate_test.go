package org

import (
	"strings"
	"testing"
)

type unknownMutateOp struct{}

func (unknownMutateOp) mutateOp() {}

func TestMutateUnknownOpEmptyUserAndPatchKind(t *testing.T) {
	noSub := false
	_, err := Mutate(EmptyDomain(), MutateRequest{
		AppendSubmission: &noSub,
		Ops:              []MutateOp{PatchOp{Entity: EntityType(""), ID: "x", Set: map[string]any{"a": 1}}},
	})
	if err == nil || !strings.Contains(err.Error(), "empty entity kind") {
		t.Fatalf("empty patch kind = %v", err)
	}
	_, err = Mutate(EmptyDomain(), MutateRequest{
		AppendSubmission: &noSub,
		Ops:              []MutateOp{unknownMutateOp{}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown mutate op") {
		t.Fatalf("unknown op = %v", err)
	}
	_, err = Mutate(EmptyDomain(), MutateRequest{
		UserID: "",
		JobID:  "intake",
		TaskID: "register",
		Ops:    []MutateOp{MintOp{Entity: EntityType("lot"), ID: "L1", Set: map[string]any{"x": 1}}},
	})
	if err == nil || !strings.Contains(err.Error(), "userId required") {
		t.Fatalf("empty user = %v", err)
	}
}

func TestMutateMintsAndAppendsSubmission(t *testing.T) {
	got, err := Mutate(EmptyDomain(), MutateRequest{
		UserID:        "ada",
		UserName:      "Ada Lovelace",
		JobID:         "intake",
		TaskID:        "register",
		JobInstanceID: "load-1",
		Timestamp:     "2026-10-22T12:00:00.000Z",
		Facts:         map[string]any{"palletCount": 2, "truckWeight": 900},
		Ops: []MutateOp{
			MintOp{Entity: EntityType("lot"), ID: "load-1", Set: map[string]any{"truckWeight": 900, "palletCount": 2}},
			MintOp{Entity: EntityType("pallet"), ID: "load-1.1", Set: map[string]any{"lotId": "load-1"}},
		},
		NewJobInstance: &JobInstance{JobID: "intake", ID: "load-1", Label: "load-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["load-1"]["truckWeight"] != 900 {
		t.Fatalf("load = %#v", got.Kind("lot"))
	}
	if len(got.Submissions) != 1 || got.Submissions[0].TaskID != "register" {
		t.Fatalf("submissions = %#v", got.Submissions)
	}
	if got.Submissions[0].JobID != "intake" || got.Submissions[0].JobInstanceID != "load-1" {
		t.Fatalf("submission keys = %#v", got.Submissions[0])
	}
	if got.Submissions[0].UserName != "Ada Lovelace" {
		t.Fatalf("userName = %#v", got.Submissions[0].UserName)
	}
	wantTouches := []EntityTouch{
		{Op: "mint", Entity: "lot", ID: "load-1"},
		{Op: "mint", Entity: "pallet", ID: "load-1.1"},
	}
	if !entityTouchesEqual(got.Submissions[0].EntitiesTouched, wantTouches) {
		t.Fatalf("entitiesTouched = %#v, want %#v", got.Submissions[0].EntitiesTouched, wantTouches)
	}
	if len(got.JobInstances) != 1 || got.JobInstances[0].ID != "load-1" {
		t.Fatalf("jobInstances = %#v", got.JobInstances)
	}
}

func TestMutateRecordsEntitiesTouchedFromMintManyAndPatchEach(t *testing.T) {
	allowed := map[string]struct{}{"lot": {}, "pallet": {}}
	tr, err := RunTransform(EmptyDomain(), RunTransformInput{
		JobID: "intake", TaskID: "register", JobInstanceID: "L1", UserID: "ada",
		Facts:   map[string]any{"L1.1": map[string]any{}, "L1.2": map[string]any{}},
		Allowed: allowed,
		Transform: []any{
			map[string]any{"op": "mint", "entity": "lot", "id": `"L1"`, "set": map[string]any{"client": `"NO"`}},
			map[string]any{"op": "mintMany", "entity": "pallet", "count": "2", "id": `"L1." & $string($index+1)`, "set": map[string]any{"lotId": `"L1"`}},
			map[string]any{"op": "patchEach", "entity": "pallet", "set": map[string]any{"ok": true}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Mutate(EmptyDomain(), tr.Request)
	if err != nil {
		t.Fatal(err)
	}
	want := []EntityTouch{
		{Op: "mint", Entity: "lot", ID: "L1"},
		{Op: "mint", Entity: "pallet", ID: "L1.1"},
		{Op: "mint", Entity: "pallet", ID: "L1.2"},
		{Op: "patch", Entity: "pallet", ID: "L1.1"},
		{Op: "patch", Entity: "pallet", ID: "L1.2"},
	}
	if len(got.Submissions) != 1 || !entityTouchesEqual(got.Submissions[0].EntitiesTouched, want) {
		t.Fatalf("entitiesTouched = %#v, want %#v", got.Submissions[0].EntitiesTouched, want)
	}

	cloned := cloneDomain(got)
	if len(cloned.Submissions[0].EntitiesTouched) == 0 {
		t.Fatal("cloneDomain must copy entitiesTouched")
	}
	cloned.Submissions[0].EntitiesTouched[0].ID = "mutated"
	if got.Submissions[0].EntitiesTouched[0].ID != "L1" {
		t.Fatal("cloneDomain must deep-copy entitiesTouched")
	}
}

func entityTouchesEqual(got, want []EntityTouch) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestMutateDuplicateID(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "lot", EntityMap{"load-1": {"client": "NO"}})
	_, err := Mutate(domain, MutateRequest{
		UserID: "ada",
		JobID:  "intake",
		TaskID: "register",
		Facts:  map[string]any{},
		Ops:    []MutateOp{MintOp{Entity: EntityType("lot"), ID: "load-1", Set: map[string]any{}}},
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
