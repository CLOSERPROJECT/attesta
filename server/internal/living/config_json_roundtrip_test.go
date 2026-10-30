package living

import (
	"encoding/json"
	"testing"
)

// JSON/Mongo OrgConfig decode must yield typed Jobs with authored Task order
// so lookupTask and mint.startingNew see register first.
func TestLookupTaskAfterJSONRoundTrip(t *testing.T) {
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

	intakeVal, ok := decoded.Jobs.Get("intake")
	if !ok {
		t.Fatal("missing intake after JSON decode")
	}
	job, typed := intakeVal.(Job)
	if !typed {
		t.Fatalf("JSON decode should store typed Job, got %T", intakeVal)
	}
	ids := OrderedTaskIDs(job.Tasks)
	if len(ids) == 0 || ids[0] != "register" {
		t.Fatalf("firstTaskId after JSON = %v, want register first", ids)
	}

	_, task, err := lookupTask(decoded, "intake", "register")
	if err != nil {
		t.Fatalf("lookupTask after JSON round-trip: %v", err)
	}
	if task == nil {
		t.Fatal("register task nil")
	}

	normalizeOrgConfigJobs(&decoded)
	intakeVal, ok = decoded.Jobs.Get("intake")
	if !ok {
		t.Fatal("missing intake after normalize")
	}
	if _, typed := intakeVal.(Job); !typed {
		t.Fatalf("normalizeOrgConfigJobs should keep Job, got %T", intakeVal)
	}
	_, _, err = lookupTask(decoded, "intake", "register")
	if err != nil {
		t.Fatalf("lookupTask after normalize: %v", err)
	}
}

func TestAsJobRejectsNil(t *testing.T) {
	if _, ok := AsJob(nil); ok {
		t.Fatal("AsJob(nil) should fail")
	}
}

func TestAsJobRejectsUnorderedMap(t *testing.T) {
	if _, ok := AsJob(map[string]any{"instanceId": "$load.id", "tasks": map[string]any{}}); ok {
		t.Fatal("AsJob(map) must not remarshal nested tasks")
	}
}
