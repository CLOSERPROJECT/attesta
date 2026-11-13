package living

import (
	"strings"
	"testing"
)

func TestFixtureMarkProgressReadsOpenLotPallets(t *testing.T) {
	line := lineJob(t)
	task, ok := TaskByID(line.Tasks, "markProgress")
	if !ok || task.IsGroup() {
		t.Fatalf("markProgress leaf = %#v", task)
	}
	dyn := task["dynamicProperties"].(map[string]any)
	progress := dyn["progress"].(map[string]any)
	if progress["read"] != "rows" {
		t.Fatalf("progress.read = %#v", progress["read"])
	}
	reads := task["reads"].(map[string]any)
	rows := reads["rows"].(map[string]any)
	where, _ := rows["where"].(string)
	if !strings.Contains(where, "$exists(weight)") {
		t.Fatalf("markProgress reads must require weight: %q", where)
	}
	if !strings.Contains(where, "$not(progress = 'done')") {
		t.Fatalf("markProgress reads must exclude progress=done: %q", where)
	}
}

func TestCommitMarkProgressPatchesPallets(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	result := commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "partial",
	})
	if result.Domain.Kind("pallet")[loadID+".1"]["progress"] != "done" {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if result.Domain.Kind("pallet")[loadID+".2"]["progress"] != "partial" {
		t.Fatalf("pallet .2 = %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
	rows := markProgressRowIDs(result.Domain, loadID)
	if len(rows) != 1 || rows[0] != loadID+".2" {
		t.Fatalf("rows after batch = %#v", rows)
	}
}

func TestCommitMarkProgressDeniesUnweighedPallets(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)

	cfg := mustActiveConfig(t, c)
	wl := ProjectWorklist(cfg, mustDomain(t, c), adminOp())
	inst, _ := worklistInstance(worklistMustJob(t, wl, "line"), lineID)
	if inst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress before weigh = %q, want unavailable", inst.Tasks["markProgress"])
	}
	if rows := markProgressRowIDs(mustDomain(t, c), loadID); len(rows) != 0 {
		t.Fatalf("unweighed rows = %#v, want none", rows)
	}

	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/markProgress",
		JobInstanceID: lineID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "partial",
				loadID + ".2": "partial",
			},
		},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("unweighed markProgress err = %v", err)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Submissions) != len(before.Submissions) {
		t.Fatal("unweighed markProgress must not append Submission")
	}
	if _, has := domain.Kind("pallet")[loadID+".1"]["progress"]; has {
		t.Fatalf("unweighed pallet must not gain progress; got %#v", domain.Kind("pallet")[loadID+".1"])
	}

	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	weighted := mustDomain(t, c)
	rows := markProgressRowIDs(weighted, loadID)
	if len(rows) != 1 || rows[0] != loadID+".1" {
		t.Fatalf("rows after one weigh = %#v, want only %s.1", rows, loadID)
	}
	afterWeigh := ProjectWorklist(cfg, weighted, adminOp())
	inst, _ = worklistInstance(worklistMustJob(t, afterWeigh, "line"), lineID)
	if inst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress after one weigh = %q, want actionable", inst.Tasks["markProgress"])
	}
	result := commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".1": "partial",
	})
	if result.Domain.Kind("pallet")[loadID+".1"]["progress"] != "partial" {
		t.Fatalf("weighted pallet progress = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if _, has := result.Domain.Kind("pallet")[loadID+".2"]["progress"]; has {
		t.Fatalf("unweighed sibling must stay without progress; got %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
}

func TestCommitMarkProgressRejectsEmptyBatch(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	_, lineID := seedLineReadyForLineLoad(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/markProgress",
		JobInstanceID: lineID,
		Value:         map[string]any{"progress": map[string]any{}},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ce.Message, "nothing to submit") {
		t.Fatalf("message = %q", ce.Message)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Submissions) != len(before.Submissions) {
		t.Fatal("empty some batch must not append Submission")
	}
}

func TestCommitMarkProgressAllowsMultiple(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	_ = commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".1": "partial",
		loadID + ".2": "partial",
	})
	second := commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
	var marks int
	for _, s := range second.Domain.Submissions {
		if s.JobID == "line" && s.TaskID == "markProgress" && s.JobInstanceID == lineID {
			marks++
		}
	}
	if marks != 2 {
		t.Fatalf("markProgress submissions = %d, want 2", marks)
	}
	wl := ProjectWorklist(mustActiveConfig(t, c), second.Domain, adminOp())
	inst, ok := worklistInstance(worklistMustJob(t, wl, "line"), lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	if inst.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress after all done = %q, want unavailable", inst.Tasks["markProgress"])
	}
}

func TestFixtureMarkProgressAdditionalPropertiesIsCellMap(t *testing.T) {
	task, _ := TaskByID(lineJob(t).Tasks, "markProgress")
	add := additionalPropertiesSchema(map[string]any(task), "progress")
	if !additionalPropertiesIsCellMap(add) {
		t.Fatalf("additionalPropertiesIsCellMap = false; schema=%#v", add)
	}
}

func markProgressRowIDs(domain Domain, openLotID string) []string {
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityType("pallet"),
		Where:  "lotId = '" + openLotID + "' and $exists(weight) and $not(progress = 'done')",
		Key:    "id",
	}, "", "", nil)
	if err != nil {
		return nil
	}
	return ids
}
