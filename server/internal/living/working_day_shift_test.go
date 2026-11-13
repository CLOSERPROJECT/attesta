package living

import (
	"context"
	"testing"
)

// readyWhenGateAuthorizer mirrors Cerbos living-commit gates without an HTTP PDP:
// deny when Unlock sequence or Task readyWhen is false.
type readyWhenGateAuthorizer struct{}

func (readyWhenGateAuthorizer) AllowCommit(_ context.Context, q AuthorizeCommit) (bool, error) {
	return q.SequenceOk && q.ReadyWhenOk, nil
}

func TestCannotOpenSecondWorkingDayWhileOneOpen(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	if _, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi"); err != nil {
		t.Fatal(err)
	}
	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	wd, ok := worklistJobByID(ProjectWorklist(cfg, domain, op), "workingDay")
	if !ok || wd.CanStart {
		t.Fatalf("workingDay canStart with one open day = %#v, want false", wd)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "workingDay/open",
		Value:    map[string]any{"date": "2026-10-20", "mode": "multi"},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("second open day must deny: %v", err)
	}
}

func TestCannotOpenSecondShiftWhileOneOpen(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	if _, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": "1st"},
		Mint:     startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	line, ok := worklistJobByID(ProjectWorklist(cfg, domain, lineOp), "line")
	if !ok || line.CanStart {
		t.Fatalf("line canStart with open shift = %#v, want false", line)
	}

	_, err = c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": "2nd"},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("second openTurno must deny: %v", err)
	}
}

func TestWholeDayOpenBindsCloseThenCloseDay(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	dayID, err := commitOpenWorkingDay(t, c, "2026-10-19", "whole-day")
	if err != nil {
		t.Fatal(err)
	}
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	shiftID := dayID + "-whole-day"
	shift, ok := domain.Kind("shift")[shiftID]
	if !ok || shift["status"] != "open" || shift["turno"] != "whole-day" {
		t.Fatalf("whole-day open must mint shift: %#v", domain.Kind("shift"))
	}

	line, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": dayID, "turno": "whole-day"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if line.JobInstanceID != shiftID {
		t.Fatalf("openTurno bind id = %q, want %q", line.JobInstanceID, shiftID)
	}
	if got := line.Domain.Kind("shift")[shiftID]["status"]; got != "open" {
		t.Fatalf("bind must keep existing shift open: %#v", line.Domain.Kind("shift"))
	}
	var shiftCount int
	for range line.Domain.Kind("shift") {
		shiftCount++
	}
	if shiftCount != 1 {
		t.Fatalf("bind must not mint a second shift: %#v", line.Domain.Kind("shift"))
	}

	cfg := mustActiveConfig(t, c)
	lineWL, ok := worklistJobByID(ProjectWorklist(cfg, line.Domain, lineOp), "line")
	if !ok || lineWL.CanStart {
		t.Fatalf("line canStart after whole-day bind = %#v, want false", lineWL)
	}
	_, err = c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": dayID, "turno": "whole-day"},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("second openTurno after whole-day bind must deny: %v", err)
	}

	if _, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath:      "line/closeTurno",
		JobInstanceID: shiftID,
		Value:         map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      "workingDay/close",
		JobInstanceID: dayID,
		Value:         map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	closed, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if closed.Kind("workingDay")[dayID]["status"] != "closed" {
		t.Fatalf("day status = %#v", closed.Kind("workingDay")[dayID])
	}
	if closed.Kind("shift")[shiftID]["status"] != "closed" {
		t.Fatalf("shift status = %#v", closed.Kind("shift")[shiftID])
	}
}

func TestIntakeUnavailableWithoutOpenDayThenSucceedsWithWorkingDayId(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	cfg := mustActiveConfig(t, c)

	emptyWL := ProjectWorklist(cfg, EmptyDomain(), admin)
	intake, ok := worklistJobByID(emptyWL, "intake")
	if !ok || intake.CanStart {
		t.Fatalf("intake canStart with no open day = %#v, want false", intake)
	}

	_, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("truckArrive without open day must deny: %v", err)
	}

	dayID, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi")
	if err != nil {
		t.Fatal(err)
	}
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	intake, ok = worklistJobByID(ProjectWorklist(cfg, domain, admin), "intake")
	if !ok || !intake.CanStart {
		t.Fatalf("intake canStart with open day = %#v, want true", intake)
	}

	arrive, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTruck := "261019-1"
	if arrive.JobInstanceID != wantTruck {
		t.Fatalf("truck id = %q, want %q", arrive.JobInstanceID, wantTruck)
	}
	truck := arrive.Domain.Kind("truck")[wantTruck]
	if truck["workingDayId"] != dayID {
		t.Fatalf("workingDayId = %#v, want %q", truck["workingDayId"], dayID)
	}
	if _, has := truck["date"]; has {
		t.Fatalf("truck must not stamp date: %#v", truck)
	}
	if _, has := truck["turno"]; has {
		t.Fatalf("truck must not stamp turno: %#v", truck)
	}

	blocked, ok := worklistJobByID(ProjectWorklist(cfg, arrive.Domain, admin), "intake")
	if !ok || blocked.CanStart {
		t.Fatalf("intake canStart with open incomplete truck = %#v, want false", blocked)
	}
	_, err = c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	var deny *CommitError
	if err == nil || !asCommitError(err, &deny) || deny.Kind != "unauthorized" {
		t.Fatalf("second truckArrive while prior incomplete must deny: %v", err)
	}
}

func TestLogisticCanStartLineOpenTurnoWhenNoOpenShift(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	if _, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi"); err != nil {
		t.Fatal(err)
	}
	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	lineWL, ok := worklistJobByID(ProjectWorklist(cfg, domain, logistic), "line")
	if !ok || !lineWL.CanStart {
		t.Fatalf("logistic canStart line with open day and no shift = %#v, want true", lineWL)
	}
	opened, err := c.Commit(t.Context(), "feragame", logistic, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened.JobInstanceID != "2026-10-19-1st" {
		t.Fatalf("jobInstanceId = %q", opened.JobInstanceID)
	}
}

func TestCloseDayUnavailableWhileShiftOpen(t *testing.T) {
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	dayID, err := commitOpenWorkingDay(t, c, "2026-10-19", "multi")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    map[string]any{"date": "2026-10-19", "turno": "1st"},
		Mint:     startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	wd, ok := worklistJobByID(ProjectWorklist(cfg, domain, admin), "workingDay")
	if !ok {
		t.Fatal("missing workingDay")
	}
	inst, ok := worklistInstance(wd, dayID)
	if !ok || inst.Tasks["close"] != TaskStatusUnavailable {
		t.Fatalf("close while shift open = %#v, want unavailable", inst)
	}

	_, err = c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      "workingDay/close",
		JobInstanceID: dayID,
		Value:         map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("close day with open shift must deny: %v", err)
	}
}
