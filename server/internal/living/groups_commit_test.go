package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func groupIntakeConfig() OrgConfig {
	arrive := TaskConfig{
		"id":    "truckArrive",
		"roles": []any{"administration"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
	}
	register := TaskConfig{
		"id":    "register",
		"roles": []any{"administration"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
	}
	lots := TaskConfig{
		"id":                           "lots",
		"kind":                         "group",
		"repeatable":                   true,
		"startLabel":                   "Nuovo lotto",
		"closeLabel":                   "Chiudi scarico",
		"closeRequiresChildSubmission": true,
		"tasks":                        []TaskConfig{register},
	}
	weight := TaskConfig{
		"id":    "weightCheck",
		"roles": []any{"administration"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
	}
	return OrgConfig{
		ID:  "group-test",
		Org: "grouporg",
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					StartLabel: "Nuovo camion",
					InstanceID: "'T1'",
					Label:      "'Truck T1'",
					Tasks:      []TaskConfig{arrive, lots, weight},
				},
			},
		},
	}
}

func groupTestCommitter(t *testing.T) (*Committer, *Loader, *MemoryDomainStore) {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := groupIntakeConfig()
	if err := configs.Put(t.Context(), "grouporg", "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "grouporg", "v1"); err != nil {
		t.Fatal(err)
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) },
	}
	loader := &Loader{Domains: domains, Configs: configs}
	return c, loader, domains
}

func TestCommitNestedSpawnSubmitCloseNext(t *testing.T) {
	c, loader, _ := groupTestCommitter(t)
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}

	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      "intake/truckArrive",
		JobInstanceID: "",
		Value:         map[string]any{},
		Mint:          startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID
	if truckID != "T1" {
		t.Fatalf("truck id = %q", truckID)
	}
	if arrive.Next != CommitNextOverview {
		t.Fatalf("after arrive next = %q, want overview (open lots)", arrive.Next)
	}

	intake, ok := worklistJobByID(arrive.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, truckID)
	if !ok {
		t.Fatal("missing instance")
	}
	if inst.Tasks["lots"] != TaskStatusAvailable {
		t.Fatalf("lots status = %q", inst.Tasks["lots"])
	}
	g, ok := inst.Groups["lots"]
	if !ok || !g.CanSpawn || g.CanClose || g.StartLabel != "Nuovo lotto" || g.CloseLabel != "Chiudi scarico" {
		t.Fatalf("lots group chrome = %#v", g)
	}

	provisional, err := loader.LoadTaskBundle(t.Context(), "grouporg", op, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
	if err != nil || provisional == nil {
		t.Fatalf("provisional nested form: %v", err)
	}
	if provisional.GroupID != "lots" || provisional.GroupInstanceID != "" || provisional.TaskID != "register" {
		t.Fatalf("provisional bundle coords = %#v", provisional)
	}

	spawned, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         map[string]any{},
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if spawned.Next != CommitNextOverview {
		t.Fatalf("spawn Commit next = %q, want overview while group open", spawned.Next)
	}
	if spawned.NextTaskFormBundle != nil {
		t.Fatal("spawn Commit must omit nextTaskFormBundle")
	}
	if len(spawned.Domain.GroupInstances) != 1 || spawned.Domain.GroupInstances[0].ID != "lots-1" {
		t.Fatalf("groupInstances = %#v", spawned.Domain.GroupInstances)
	}
	sub := spawned.Domain.Submissions[len(spawned.Domain.Submissions)-1]
	if sub.GroupID != "lots" || sub.GroupInstanceID != "lots-1" || sub.TaskID != "register" {
		t.Fatalf("nested submission = %#v", sub)
	}
	intake, _ = worklistJobByID(spawned.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	g = inst.Groups["lots"]
	if !g.CanSpawn || !g.CanClose || len(g.Instances) != 1 {
		t.Fatalf("after spawn+submit lots = %#v", g)
	}
	if g.Instances[0].Tasks["register"] != TaskStatusDone {
		t.Fatalf("register status = %#v", g.Instances[0].Tasks)
	}
	if inst.Tasks["weightCheck"] != TaskStatusLocked {
		t.Fatalf("weightCheck before close = %q", inst.Tasks["weightCheck"])
	}

	closed, err := c.CloseGroup(t.Context(), "grouporg", op, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Next != CommitNextTask {
		t.Fatalf("after close next = %q, want task (weightCheck)", closed.Next)
	}
	if closed.NextTaskFormBundle == nil || closed.NextTaskFormBundle.TaskID != "weightCheck" {
		t.Fatalf("after close nextTaskFormBundle = %#v", closed.NextTaskFormBundle)
	}
	intake, _ = worklistJobByID(closed.Worklist, "intake")
	inst, _ = worklistInstance(intake, truckID)
	if inst.Tasks["lots"] != TaskStatusDone {
		t.Fatalf("lots after close = %q", inst.Tasks["lots"])
	}
	if inst.Groups["lots"].CanSpawn || inst.Groups["lots"].CanClose {
		t.Fatalf("closed group affordances = %#v", inst.Groups["lots"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck after close = %q, want actionable", inst.Tasks["weightCheck"])
	}
}

func TestDomainSpawnGroupInstanceThenCommitLeaf(t *testing.T) {
	c, _, domains := groupTestCommitter(t)
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}
	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID

	opened, err := SpawnGroupInstance(arrive.Domain, "intake", truckID, "lots", "lots-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), "grouporg", opened); err != nil {
		t.Fatal(err)
	}

	committed, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", "lots-1", "register"),
		JobInstanceID: truckID,
		Value:         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.Next != CommitNextOverview {
		t.Fatalf("nested leaf Commit next = %q, want overview", committed.Next)
	}
	sub := committed.Domain.Submissions[len(committed.Domain.Submissions)-1]
	if sub.GroupInstanceID != "lots-1" {
		t.Fatalf("submission = %#v", sub)
	}
}

func TestCommitErrorFromGroupErrorMissingChild(t *testing.T) {
	ce := commitErrorFromGroupError(&GroupError{
		Kind:    "missing-child-submission",
		Message: "living: cannot close lots",
	})
	if ce.Kind != "missing-child-submission" {
		t.Fatalf("kind = %q", ce.Kind)
	}
	if ce.Message != "living: cannot close lots" {
		t.Fatalf("message = %q", ce.Message)
	}
	ce = commitErrorFromGroupError(&GroupError{Kind: "already-closed", Message: "closed"})
	if ce.Kind != "prep" {
		t.Fatalf("other group kinds map to prep, got %q", ce.Kind)
	}
}

func TestCloseHTTPAndNestedTaskForm(t *testing.T) {
	c, loader, _ := groupTestCommitter(t)
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}
	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID
	if _, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         map[string]any{},
		Mint:          spawningNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return op, true
		},
	}

	formReq := httptest.NewRequest(http.MethodGet,
		"/api/living/task-form?jobId=intake&taskId=register&jobInstanceId="+truckID+
			"&groupId=lots&groupInstanceId="+ProvisionalGroupInstanceID, nil)
	formRec := httptest.NewRecorder()
	h.ServeHTTP(formRec, formReq)
	// After spawn, provisional can still spawn another — mount OK.
	if formRec.Code != http.StatusOK {
		t.Fatalf("provisional task-form status = %d body=%s", formRec.Code, formRec.Body.String())
	}

	body := `{"jobId":"intake","jobInstanceId":"` + truckID + `","groupId":"lots"}`
	req := httptest.NewRequest(http.MethodPost, "/api/living/close", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("close status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["next"] != string(CommitNextTask) {
		t.Fatalf("close JSON next = %#v", wire["next"])
	}
	if _, ok := wire["nextTaskFormBundle"].(map[string]any); !ok {
		t.Fatalf("close JSON missing nextTaskFormBundle: %s", rec.Body.String())
	}
	if _, has := wire["taskFormBundles"]; has {
		t.Fatal("close must not ship bulk taskFormBundles")
	}
}
