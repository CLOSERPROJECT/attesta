package living

import (
	"strings"
	"testing"
)

func weighRow(weight float64) map[string]any {
	return map[string]any{
		"weight": weight, "tareKind": "euro", "idoneita": "NIR", "panelCount": 1.0,
	}
}

func palletWeightValue(weights map[string]any) map[string]any {
	return map[string]any{
		"averagePanelWeight": 10.0,
		"tareKind":           "euro",
		"weights":            weights,
	}
}

func commitThroughCodiceFir(t *testing.T, c *Committer, instanceID string) {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration", "logistic"}}
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-1"},
	}); err != nil {
		t.Fatal(err)
	}
}

func intakeTasks(t *testing.T) Job {
	t.Helper()
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Jobs.Values["intake"].(Job)
}

func lineJob(t *testing.T) Job {
	t.Helper()
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Jobs.Values["line"].(Job)
}

func palletWeightTask(intake Job) any {
	v, _ := intake.Tasks.Get("palletWeight")
	return v
}

func TestCommitPalletWeightRejectsPartialLeftoversAll(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: palletWeightValue(map[string]any{
			instanceID + ".1": weighRow(11),
			// .2 omitted — incomplete all
		}),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ce.Message, "completeness all") {
		t.Fatalf("message = %q", ce.Message)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Submissions) != 2 {
		t.Fatalf("partial must not append Submission: %#v", domain.Submissions)
	}
	if _, has := domain.Pallet[instanceID+".1"]["weight"]; has {
		t.Fatal("partial reject must not patch pallets")
	}

	intake := intakeTasks(t)
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(intake.Tasks), JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: instanceID, Submissions: domain.Submissions, Tasks: intake.Tasks,
	}) {
		t.Fatal("incomplete all must not unlock weightCheck")
	}
	line := lineJob(t)
	if IsStartsWhenSatisfied(line.StartsWhen, domain.Submissions, domain.JobInstances, palletWeightTask(intake)) {
		t.Fatal("incomplete all must not satisfy line startsWhen")
	}
}

func TestCommitPalletWeightCompleteAllUnlocksAndStartsWhen(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: palletWeightValue(map[string]any{
			instanceID + ".1": weighRow(11),
			instanceID + ".2": weighRow(22),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Pallet[instanceID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", result.Domain.Pallet[instanceID+".1"])
	}
	if result.Domain.Pallet[instanceID+".2"]["weight"] != 22.0 {
		t.Fatalf("pallet .2 = %#v", result.Domain.Pallet[instanceID+".2"])
	}
	if len(result.Domain.Submissions) != 3 || result.Domain.Submissions[2].TaskID != "palletWeight" {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
	if leftovers := palletWeightLeftoverIDs(result.Domain, instanceID); len(leftovers) != 0 {
		t.Fatalf("leftovers after complete = %#v", leftovers)
	}

	intake := intakeTasks(t)
	if !taskMayBeUnlockGate(palletWeightTask(intake)) {
		t.Fatal("completeness all must remain an unlock / startsWhen gate")
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(intake.Tasks), JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: instanceID, Submissions: result.Domain.Submissions, Tasks: intake.Tasks,
	}) {
		t.Fatal("completed all must unlock weightCheck")
	}
	line := lineJob(t)
	if !IsStartsWhenSatisfied(line.StartsWhen, result.Domain.Submissions, result.Domain.JobInstances, palletWeightTask(intake)) {
		t.Fatal("completed all must enable authored line startsWhen target")
	}
}

func TestIsTaskUnlockedAfterCompletenessAllSubmission(t *testing.T) {
	tasks := intakeUnlockTasks()
	order := []string{"register", "codiceFir", "palletWeight", "weightCheck"}
	afterFir := []Submission{
		sub("intake", "register", "load-1"),
		sub("intake", "codiceFir", "load-1"),
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "load-1", Submissions: afterFir, Tasks: tasks,
	}) {
		t.Fatal("weightCheck locked until palletWeight Submission")
	}
	afterAll := append(afterFir, sub("intake", "palletWeight", "load-1"))
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "load-1", Submissions: afterAll, Tasks: tasks,
	}) {
		t.Fatal("palletWeight completeness all Submission unlocks weightCheck")
	}
}
