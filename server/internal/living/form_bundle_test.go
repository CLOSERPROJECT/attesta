package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func bundleAddress(b *TaskFormBundle) string {
	if b == nil {
		return "<nil>"
	}
	return b.JobID + "/" + b.TaskID + "@" + b.JobInstanceID
}

func findBundle(bundles []*TaskFormBundle, jobID, taskID, instanceID string) *TaskFormBundle {
	for _, b := range bundles {
		if b != nil && b.JobID == jobID && b.TaskID == taskID && b.JobInstanceID == instanceID {
			return b
		}
	}
	return nil
}

func TestLoadCommitJSONOmitsDomainAndConfig(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	loaded, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	loadRaw, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	var loadWire map[string]any
	if err := json.Unmarshal(loadRaw, &loadWire); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"domain", "config"} {
		if _, has := loadWire[key]; has {
			t.Fatalf("Load JSON must omit %q", key)
		}
	}
	if loadWire["configId"] != "truck-loop" {
		t.Fatalf("configId = %#v", loadWire["configId"])
	}
	if _, ok := loadWire["worklist"].(map[string]any); !ok {
		t.Fatal("Load JSON missing worklist")
	}
	if _, ok := loadWire["taskFormBundles"].([]any); !ok {
		t.Fatal("Load JSON missing taskFormBundles")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	committed, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
		Mint:          startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	commitRaw, err := json.Marshal(committed)
	if err != nil {
		t.Fatal(err)
	}
	var commitWire map[string]any
	if err := json.Unmarshal(commitRaw, &commitWire); err != nil {
		t.Fatal(err)
	}
	if _, has := commitWire["domain"]; has {
		t.Fatal("Commit JSON must omit domain")
	}
	if _, has := commitWire["config"]; has {
		t.Fatal("Commit JSON must omit config")
	}
	if _, ok := commitWire["worklist"].(map[string]any); !ok {
		t.Fatal("Commit JSON missing worklist")
	}
	if _, ok := commitWire["taskFormBundles"].([]any); !ok {
		t.Fatal("Commit JSON missing taskFormBundles")
	}
}

func TestLoadJSONTaskFormBundlesProvisionalWhenCanStart(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	result, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok || !intake.CanStart {
		t.Fatalf("intake canStart = %#v", intake)
	}
	if len(result.TaskFormBundles) != 1 {
		t.Fatalf("bundles = %#v, want provisional register only", result.TaskFormBundles)
	}
	b := result.TaskFormBundles[0]
	if b.JobID != "intake" || b.TaskID != "register" || b.JobInstanceID != "" {
		t.Fatalf("provisional = %s", bundleAddress(b))
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	arr, ok := wire["taskFormBundles"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("Load JSON taskFormBundles = %#v", wire["taskFormBundles"])
	}
	row, _ := arr[0].(map[string]any)
	if row["taskId"] != "register" || row["jobInstanceId"] != "" {
		t.Fatalf("wire provisional = %#v", row)
	}
}

func TestLoadJSONTaskFormBundlesAllAvailablePlusProvisional(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: instanceID,
		Value:         registerValue(),
		Mint:          startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	result, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	provisional := findBundle(result.TaskFormBundles, "intake", "register", "")
	if provisional == nil {
		t.Fatal("missing provisional register bundle")
	}
	codice := findBundle(result.TaskFormBundles, "intake", "codiceFir", instanceID)
	if codice == nil {
		t.Fatalf("missing codiceFir bundle; got %#v", addresses(result.TaskFormBundles))
	}
	if codice.Surface != "live" || codice.Schema == nil || codice.Bindings == nil {
		t.Fatalf("codiceFir incomplete: %#v", codice)
	}
	if findBundle(result.TaskFormBundles, "intake", "register", instanceID) != nil {
		t.Fatal("done register must not have an instance bundle")
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	arr, ok := wire["taskFormBundles"].([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("Load JSON taskFormBundles = %#v", wire["taskFormBundles"])
	}
}

func addresses(bundles []*TaskFormBundle) []string {
	out := make([]string, 0, len(bundles))
	for _, b := range bundles {
		out = append(out, bundleAddress(b))
	}
	return out
}

func TestLoadJSONTaskFormBundlePalletWeightLeftovers(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	result, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	bundle := findBundle(result.TaskFormBundles, "intake", "palletWeight", instanceID)
	if bundle == nil {
		t.Fatalf("missing palletWeight; got %#v", addresses(result.TaskFormBundles))
	}
	if bundle.Surface != "live" {
		t.Fatalf("surface = %q", bundle.Surface)
	}
	if len(bundle.LeftoverIDs) == 0 {
		t.Fatal("palletWeight bundle missing leftoverIds")
	}
	if bundle.ProjectionKey == "" {
		t.Fatal("missing projectionKey")
	}
	if len(bundle.Derived) == 0 {
		t.Fatal("expected expanded derived rules")
	}
	domainBag, _ := bundle.Bindings["domain"].(map[string]any)
	if domainBag["tareKinds"] == nil {
		t.Fatal("bindings.domain missing tareKinds")
	}
	for _, key := range []string{"load", "pallet", "container"} {
		if _, ok := domainBag[key].(map[string]any); !ok {
			t.Fatalf("bindings.domain.%s = %#v, want object", key, domainBag[key])
		}
	}
	subs, ok := domainBag["submissions"].([]any)
	if !ok {
		t.Fatalf("bindings.domain.submissions = %#v, want array", domainBag["submissions"])
	}
	if len(subs) == 0 {
		t.Fatal("expected submissions after codiceFir for DomainSnapshot-shaped bindings")
	}
	weights, _ := bundle.InitialData["weights"].(map[string]any)
	if len(weights) != len(bundle.LeftoverIDs) {
		t.Fatalf("initial weights = %#v, leftovers = %#v", weights, bundle.LeftoverIDs)
	}
}

func TestHandleLoadJSONIncludesTaskFormBundles(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: instanceID,
		Value:         registerValue(),
		Mint:          startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Loader: loader,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return op, true
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	arr, ok := wire["taskFormBundles"].([]any)
	if !ok {
		t.Fatalf("HTTP Load missing taskFormBundles: %s", rec.Body.String())
	}
	found := false
	for _, item := range arr {
		row, _ := item.(map[string]any)
		if row["taskId"] == "codiceFir" && row["jobInstanceId"] == instanceID {
			found = true
		}
	}
	if !found {
		t.Fatalf("HTTP Load missing codiceFir bundle: %#v", arr)
	}
}

func TestCommitReplacesTaskFormBundles(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: instanceID,
		Value:         registerValue(),
		Mint:          startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if findBundle(loaded.TaskFormBundles, "intake", "codiceFir", instanceID) == nil {
		t.Fatalf("pre-commit bundles = %#v", addresses(loaded.TaskFormBundles))
	}

	committed, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/codiceFir",
		JobInstanceID: instanceID,
		Value:         map[string]any{"codiceFir": "FIR-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if findBundle(committed.TaskFormBundles, "intake", "codiceFir", instanceID) != nil {
		t.Fatal("Commit must drop codiceFir bundle after it leaves available")
	}
	pallet := findBundle(committed.TaskFormBundles, "intake", "palletWeight", instanceID)
	if pallet == nil {
		t.Fatalf("Commit missing palletWeight; got %#v", addresses(committed.TaskFormBundles))
	}
	if findBundle(committed.TaskFormBundles, "intake", "register", "") == nil {
		t.Fatal("Commit must keep provisional register while canStart")
	}

	raw, err := json.Marshal(committed)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["taskFormBundles"].([]any); !ok {
		t.Fatalf("Commit JSON missing taskFormBundles: %s", string(raw))
	}
}

func TestTruckLoopCommitsWalkBundlesIntakeAndLine(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	opLine := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	intakeID := "NIR-261019-NO-1"

	completeIntakeThroughPalletWeight(t, c, intakeID)

	shift, err := c.Commit(t.Context(), "feragame", opLine, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: "1st", Value: map[string]any{"shift": "1st"}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if findBundle(shift.TaskFormBundles, "intake", "register", "") == nil {
		t.Fatal("intake provisional register missing after line shift")
	}
	if findBundle(shift.TaskFormBundles, "line", "shift", "") == nil {
		t.Fatal("line canStart provisional shift missing after first shift mint")
	}
	lineLoad := findBundle(shift.TaskFormBundles, "line", "lineLoad", "1st")
	if lineLoad == nil {
		t.Fatalf("after shift: %#v", addresses(shift.TaskFormBundles))
	}

	loaded, err := c.Commit(t.Context(), "feragame", opLine, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if findBundle(loaded.TaskFormBundles, "line", "lineLoad", "1st") == nil {
		t.Fatalf("lineLoad still open should keep bundle; got %#v", addresses(loaded.TaskFormBundles))
	}
	if findBundle(loaded.TaskFormBundles, "line", "containerReg", "1st") == nil {
		t.Fatalf("containerReg available missing; got %#v", addresses(loaded.TaskFormBundles))
	}
}
