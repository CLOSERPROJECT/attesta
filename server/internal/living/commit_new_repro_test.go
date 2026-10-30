package living

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// After Mongo/JSON OrgConfig decode, intake Task order must stay authored
// (register first). Remarshaling Jobs via unordered map sorted tasks to
// codiceFir first → Nuovo camion minted on the wrong Task → empty $load.id.
func TestSmokeNuovoCamionInstanceIdAfterMongoJSON(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OrgConfig
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	normalizeOrgConfigJobs(&decoded)

	intake, _ := decoded.Jobs.Get("intake")
	job, ok := AsJob(intake)
	if !ok {
		t.Fatalf("intake Job after JSON: %T", intake)
	}
	ids := OrderedTaskIDs(job.Tasks)
	if len(ids) == 0 || ids[0] != "register" {
		t.Fatalf("firstTaskId after Mongo JSON = %v, want register first", ids)
	}

	configs := NewMemoryConfigStore()
	if err := configs.Put(t.Context(), "feragame", decoded.Version, decoded); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "feragame", decoded.Version); err != nil {
		t.Fatal(err)
	}

	c := &Committer{
		Domains:    NewMemoryDomainStore(),
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register",
		Value:    registerValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != "NIR-261019-NO-1" {
		t.Fatalf("jobInstanceId=%q want NIR-261019-NO-1", result.JobInstanceID)
	}
	for _, inst := range result.Domain.JobInstances {
		if inst.ID == "_new" {
			t.Fatalf("Domain must not contain _new sentinel: %#v", result.Domain.JobInstances)
		}
	}
}

func TestCommitRegisterStartingNewAfterMongoJSONMintsLoadID(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OrgConfig
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	normalizeOrgConfigJobs(&decoded)

	intake, _ := decoded.Jobs.Get("intake")
	job, ok := AsJob(intake)
	if !ok {
		t.Fatalf("intake Job after JSON: %T", intake)
	}
	ids := OrderedTaskIDs(job.Tasks)
	if len(ids) == 0 || ids[0] != "register" {
		t.Fatalf("OrderedTaskIDs after JSON normalize = %v, want register first", ids)
	}

	configs := NewMemoryConfigStore()
	_ = configs.Put(t.Context(), "feragame", decoded.Version, decoded)
	_ = configs.SetActiveVersion(t.Context(), "feragame", decoded.Version)
	c := &Committer{
		Domains: NewMemoryDomainStore(), Configs: configs, Authorizer: AllowAllAuthorizer{},
		Now: func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", Value: registerValue(), Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != "NIR-261019-NO-1" {
		t.Fatalf("jobInstanceId=%q want NIR-261019-NO-1", result.JobInstanceID)
	}
}

func TestCommitRejectsWireNewSentinel(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "_new",
		Value:         registerValue(),
		Mint:          startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ce.Message, "_new") {
		t.Fatalf("prep message should mention _new: %q", ce.Message)
	}
}
