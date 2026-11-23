package org

import (
	"testing"
)

// Same client + same calendar day must mint …-1 then …-2. Lot id lives on the
// Domain map key only; counting $v.id always reminted -1 (duplicate-id).
func TestIntakeRegisterSameClientSameDayIncrementsProgressive(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := plantCommitArrive(t, c)

	first := plantCommitRegister(t, c, truckID, registerLotValue())
	load1 := firstLoadID(first.Domain)
	if load1 != "CO-261029-1" {
		t.Fatalf("first lot id = %q, want CO-261029-1", load1)
	}
	prog1 := first.Domain.Kind("lot")[load1]["clientProgressive"]
	if prog1 != float64(1) && prog1 != 1 {
		t.Fatalf("first clientProgressive = %#v, want 1", prog1)
	}

	lot2 := registerLotValue()
	lot2["numeroFir"] = "FIR-2"
	lot2["truckWeightAfter"] = 400.0
	second, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         lot2,
	})
	if err != nil {
		t.Fatalf("second same-client register: %v", err)
	}
	load2 := ""
	for id := range second.Domain.Kind("lot") {
		if id != load1 {
			load2 = id
			break
		}
	}
	if load2 != "CO-261029-2" {
		t.Fatalf("second lot id = %q, want CO-261029-2", load2)
	}
	prog2 := second.Domain.Kind("lot")[load2]["clientProgressive"]
	if prog2 != float64(2) && prog2 != 2 {
		t.Fatalf("second clientProgressive = %#v, want 2", prog2)
	}
}
