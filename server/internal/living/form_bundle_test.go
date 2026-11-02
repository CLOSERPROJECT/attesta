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
	loaded, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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
	if loadWire["configId"] != "feragame-config" {
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
	result, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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

	result, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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

func TestLoadJSONTaskFormBundlePalletWeightRows(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	result, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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
	if len(bundle.RowIDs) == 0 {
		t.Fatal("palletWeight bundle missing rowIds")
	}
	if bundle.ProjectionKey == "" {
		t.Fatal("missing projectionKey")
	}
	if len(bundle.Derived) == 0 {
		t.Fatal("expected expanded derived rules")
	}
	domainBag, _ := bundle.Bindings["domain"].(map[string]any)
	if _, has := domainBag["tareKinds"]; has {
		t.Fatal("bindings.domain must not inject tareKinds")
	}
	catalogsBag, _ := bundle.Bindings["catalogs"].(map[string]any)
	if catalogsBag == nil {
		t.Fatal("bindings.catalogs missing")
	}
	tare, _ := catalogsBag["tareKinds"].(map[string]any)
	if tare["euro"] == nil {
		t.Fatalf("bindings.catalogs.tareKinds = %#v", catalogsBag["tareKinds"])
	}
	if load, ok := domainBag["load"].(map[string]any); !ok || len(load) == 0 {
		t.Fatalf("bindings.domain.load = %#v, want minted load kind", domainBag["load"])
	}
	subs, ok := domainBag["submissions"].([]any)
	if !ok {
		t.Fatalf("bindings.domain.submissions = %#v, want array", domainBag["submissions"])
	}
	if len(subs) == 0 {
		t.Fatal("expected submissions after codiceFir for DomainSnapshot-shaped bindings")
	}
	if weights, has := bundle.InitialData["weights"]; has {
		m, _ := weights.(map[string]any)
		if len(m) != 0 {
			t.Fatalf("object-map initialData must not densify; got %#v", weights)
		}
	}
	props, _ := bundle.Schema["properties"].(map[string]any)
	weightsSchema, _ := props["weights"].(map[string]any)
	assertConcreteDynamicMap(t, weightsSchema, bundle.RowIDs, true)
}

func assertConcreteDynamicMap(t *testing.T, mapSchema map[string]any, rowIDs []string, fullMap bool) {
	t.Helper()
	if mapSchema == nil {
		t.Fatal("map schema missing")
	}
	if _, has := mapSchema["propertyNames"]; has {
		t.Fatalf("must not stamp propertyNames; got %#v", mapSchema["propertyNames"])
	}
	if mapSchema["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %#v, want false", mapSchema["additionalProperties"])
	}
	concrete, _ := mapSchema["properties"].(map[string]any)
	if len(concrete) != len(rowIDs) {
		t.Fatalf("properties = %#v, rows = %#v", concrete, rowIDs)
	}
	for _, id := range rowIDs {
		if _, ok := concrete[id].(map[string]any); !ok {
			t.Fatalf("missing concrete property %q in %#v", id, concrete)
		}
	}
	var requiredLen int
	switch req := mapSchema["required"].(type) {
	case []string:
		requiredLen = len(req)
	case []any:
		requiredLen = len(req)
	}
	if fullMap {
		if requiredLen != len(rowIDs) {
			t.Fatalf("full-map required = %#v, rows = %#v", mapSchema["required"], rowIDs)
		}
		return
	}
	if _, has := mapSchema["required"]; has {
		t.Fatalf("partial-map must not project required; got %#v", mapSchema["required"])
	}
}

func TestFullMapProjectsRequiredPartialMapDoesNot(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	opLogistic := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	opLine := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}

	commitThroughCodiceFir(t, c, intakeID)
	afterFir, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	pallet := findBundle(afterFir.TaskFormBundles, "intake", "palletWeight", intakeID)
	if pallet == nil {
		t.Fatalf("missing palletWeight; got %#v", addresses(afterFir.TaskFormBundles))
	}
	if len(pallet.RowIDs) == 0 {
		t.Fatal("palletWeight full-map needs Domain-read row ids")
	}
	weightsProps, _ := pallet.Schema["properties"].(map[string]any)
	weightsSchema, _ := weightsProps["weights"].(map[string]any)
	assertConcreteDynamicMap(t, weightsSchema, pallet.RowIDs, true)

	if _, err := c.Commit(t.Context(), "feragame", opLogistic, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: intakeID,
		Value: palletWeightValue(map[string]any{
			intakeID + ".1": weighRow(11),
			intakeID + ".2": weighRow(22),
		}),
	}); err != nil {
		t.Fatal(err)
	}
	shift, err := c.Commit(t.Context(), "feragame", opLine, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: "1st", Value: map[string]any{"shift": "1st"}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	lineLoad := findBundle(shift.TaskFormBundles, "line", "lineLoad", "1st")
	if lineLoad == nil {
		t.Fatalf("missing lineLoad; got %#v", addresses(shift.TaskFormBundles))
	}
	if len(lineLoad.RowIDs) == 0 {
		t.Fatal("lineLoad partial-map still projects Domain-read row ids")
	}
	progressProps, _ := lineLoad.Schema["properties"].(map[string]any)
	progressSchema, _ := progressProps["progress"].(map[string]any)
	assertConcreteDynamicMap(t, progressSchema, lineLoad.RowIDs, false)
	assertLineLoadCellMapBundle(t, lineLoad)
}

// Cell-map Tasks mount as live ConfigForm; enums stay on concrete cell schemas.
func assertLineLoadCellMapBundle(t *testing.T, bundle *TaskFormBundle) {
	t.Helper()
	if bundle.Surface != "live" {
		t.Fatalf("surface = %q, want live", bundle.Surface)
	}
	props, _ := bundle.Schema["properties"].(map[string]any)
	progress, _ := props["progress"].(map[string]any)
	assertConcreteDynamicMap(t, progress, bundle.RowIDs, false)
	concrete, _ := progress["properties"].(map[string]any)
	propUi, _ := bundle.UiSchema["progress"].(map[string]any)
	if propUi == nil {
		t.Fatal("progress uiSchema missing")
	}
	wantEnums := []string{"partial", "done"}
	for _, id := range bundle.RowIDs {
		cell, _ := concrete[id].(map[string]any)
		rawEnum, _ := cell["enum"].([]any)
		if len(rawEnum) != 2 || rawEnum[0] != "partial" || rawEnum[1] != "done" {
			t.Fatalf("cell %q enum = %#v, want [partial done] (no leading null)", id, rawEnum)
		}
		if _, hasAnyOf := cell["anyOf"]; hasAnyOf {
			t.Fatalf("cell %q must use enum not anyOf; got %#v", id, cell)
		}
		fromSchema := schemaStringEnum(cell)
		if len(fromSchema) != len(wantEnums) {
			t.Fatalf("schema enums for %q = %#v, want %#v", id, fromSchema, wantEnums)
		}
		for i, want := range wantEnums {
			if fromSchema[i] != want {
				t.Fatalf("schema enums for %q = %#v, want %#v", id, fromSchema, wantEnums)
			}
		}
		cellUi, _ := propUi[id].(map[string]any)
		cellOpts, _ := cellUi["ui:options"].(map[string]any)
		if cellOpts == nil {
			t.Fatalf("cell %q ui:options missing", id)
		}
		if _, has := cellOpts["clearable"]; has {
			t.Fatalf("cell %q must not stamp clearable; got %#v", id, cellOpts["clearable"])
		}
		if cellOpts["placeholder"] != "–" {
			t.Fatalf("cell %q placeholder = %#v, want –", id, cellOpts["placeholder"])
		}
		shadcn, _ := cellOpts["shadcn4Select"].(map[string]any)
		if shadcn["placeholder"] != "–" {
			t.Fatalf("cell %q shadcn4Select.placeholder = %#v, want –", id, shadcn["placeholder"])
		}
		names := schemaStringList(cellOpts["enumNames"])
		if len(names) != 2 || names[0] != "Parziale" || names[1] != "Fatto" {
			t.Fatalf("cell %q enumNames = %#v, want [Parziale Fatto]", id, cellOpts["enumNames"])
		}
	}
	comps, _ := propUi["ui:components"].(map[string]any)
	if comps["objectField"] != "mapTableField" {
		t.Fatalf("ui:components = %#v, want mapTableField objectField", comps)
	}
	if _, has := propUi["additionalProperties"]; has {
		t.Fatalf("uiSchema must stamp concrete ids, not additionalProperties; got %#v", propUi["additionalProperties"])
	}
	opts, _ := propUi["ui:options"].(map[string]any)
	if opts == nil {
		t.Fatal("progress ui:options missing")
	}
	cols, _ := opts["columns"].(map[string]any)
	if cols["id"] != "Bancale" || cols["value"] != "Avanzamento" {
		t.Fatalf("columns clobbered: %#v", cols)
	}
	if _, has := opts["valueType"]; has {
		t.Fatalf("ui:options must not stamp valueType; got %#v", opts["valueType"])
	}
	if _, has := opts["progressEnums"]; has {
		t.Fatalf("ui:options must not stamp progressEnums; got %#v", opts["progressEnums"])
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
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	all := FixtureAllRolesOp()
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: instanceID,
		Value:         registerValue(),
		Mint:          startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.Load(t.Context(), "feragame", all)
	if err != nil {
		t.Fatal(err)
	}
	if findBundle(loaded.TaskFormBundles, "intake", "codiceFir", instanceID) == nil {
		t.Fatalf("pre-commit bundles = %#v", addresses(loaded.TaskFormBundles))
	}

	committed, err := c.Commit(t.Context(), "feragame", all, CommitRequest{
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

func TestFeragameConfigCommitsWalkBundlesIntakeAndLine(t *testing.T) {
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
	if findBundle(shift.TaskFormBundles, "intake", "register", "") != nil {
		t.Fatal("line viewer must not get intake provisional register (administration-only)")
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
	if findBundle(loaded.TaskFormBundles, "line", "lineLoad", "1st") != nil {
		t.Fatalf("lineLoad done after Submission must drop bundle; got %#v", addresses(loaded.TaskFormBundles))
	}
	// containerReg is logistic-only; line Worklist still shows available status for handoff.
	if findBundle(loaded.TaskFormBundles, "line", "containerReg", "1st") != nil {
		t.Fatalf("line viewer must not get containerReg bundle; got %#v", addresses(loaded.TaskFormBundles))
	}
	lineJob, ok := worklistJobByID(loaded.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist")
	}
	inst, ok := worklistInstance(lineJob, "1st")
	if !ok || inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg status = %#v, want available", inst)
	}
}
