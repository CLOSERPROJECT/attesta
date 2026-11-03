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
	arr, ok := loadWire["taskFormBundles"].([]any)
	if !ok {
		t.Fatal("Load JSON missing taskFormBundles")
	}
	if len(arr) != 0 {
		t.Fatalf("home Load must omit bulk taskFormBundles, got %#v", arr)
	}
	roles, _ := loadWire["roles"].([]any)
	if len(roles) == 0 {
		t.Fatalf("Load JSON missing viewer roles: %#v", loadWire["roles"])
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	committed, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
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
	if _, has := commitWire["taskFormBundles"]; has {
		t.Fatal("Commit JSON must omit taskFormBundles")
	}
}

func TestLoadJSONOmitsBulkTaskFormBundlesWhenCanStart(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	op := FixtureAllRolesOp()
	result, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok || !intake.CanStart {
		t.Fatalf("intake canStart = %#v", intake)
	}
	if len(result.TaskFormBundles) != 0 {
		t.Fatalf("home Load bundles = %#v, want empty (lazy task-form)", result.TaskFormBundles)
	}
	if len(result.Roles) == 0 {
		t.Fatal("Load must ship viewer roles for Available affordances")
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
	if !ok || len(arr) != 0 {
		t.Fatalf("Load JSON taskFormBundles = %#v, want []", wire["taskFormBundles"])
	}
}

func TestLoadOmitsBulkWhileAddressedFetchServesAvailable(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	all := FixtureAllRolesOp()
	truckID := commitTruckArrive(t, c)

	result, err := loader.Load(t.Context(), "feragame", all)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TaskFormBundles) != 0 {
		t.Fatalf("home Load must omit bulk bundles; got %#v", addresses(result.TaskFormBundles))
	}

	provisional, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
	if err != nil || provisional == nil {
		t.Fatalf("provisional lots/register: %v %#v", err, provisional)
	}
	arrive, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "truckArrive", "", "", "")
	if err != nil || arrive == nil {
		t.Fatalf("provisional truckArrive: %v %#v", err, arrive)
	}
	if arrive.Surface != "live" || arrive.Schema == nil {
		t.Fatalf("truckArrive incomplete: %#v", arrive)
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "truckArrive", truckID, "", ""); err == nil {
		t.Fatal("done truckArrive must be forbidden on addressed fetch")
	}
}

func addresses(bundles []*TaskFormBundle) []string {
	out := make([]string, 0, len(bundles))
	for _, b := range bundles {
		out = append(out, bundleAddress(b))
	}
	return out
}

func TestLoadTaskBundlePalletWeightRows(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	home, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.TaskFormBundles) != 0 {
		t.Fatalf("home Load must omit bulk; got %#v", addresses(home.TaskFormBundles))
	}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", FixtureAllRolesOp(), "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID)
	if err != nil || bundle == nil {
		t.Fatalf("missing weigh: %v", err)
	}
	if bundle.Surface != "live" {
		t.Fatalf("surface = %q", bundle.Surface)
	}
	if len(bundle.RowIDs) == 0 {
		t.Fatal("weigh bundle missing rowIds")
	}
	found := false
	for _, id := range bundle.RowIDs {
		if id == loadID+".1" || id == loadID+".2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("rowIds = %#v, want pallets for %s", bundle.RowIDs, loadID)
	}
	if bundle.ProjectionKey == "" {
		t.Fatal("missing projectionKey")
	}
	if len(bundle.Derived) == 0 {
		t.Fatal("expected expanded derived rules")
	}
	domainBound, _ := bundle.Bindings["domain"].(map[string]any)
	if _, has := domainBound["tareKinds"]; has {
		t.Fatal("bindings.domain must not inject tareKinds")
	}
	catalogsBound, _ := bundle.Bindings["catalogs"].(map[string]any)
	if catalogsBound == nil {
		t.Fatal("bindings.catalogs missing")
	}
	tare, _ := catalogsBound["tareKinds"].(map[string]any)
	if tare["euro"] == nil {
		t.Fatalf("bindings.catalogs.tareKinds = %#v", catalogsBound["tareKinds"])
	}
	if load, ok := domainBound["load"].(map[string]any); !ok || len(load) == 0 {
		t.Fatalf("bindings.domain.load = %#v, want minted load kind", domainBound["load"])
	}
	subs, ok := domainBound["submissions"].([]any)
	if !ok || len(subs) == 0 {
		t.Fatalf("bindings.domain.submissions = %#v", domainBound["submissions"])
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
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	weigh, err := loader.LoadTaskBundle(t.Context(), "feragame", FixtureAllRolesOp(), "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID)
	if err != nil || weigh == nil {
		t.Fatalf("missing weigh: %v", err)
	}
	if len(weigh.RowIDs) == 0 {
		t.Fatal("weigh full-map needs Domain-read row ids")
	}
	weightsProps, _ := weigh.Schema["properties"].(map[string]any)
	weightsSchema, _ := weightsProps["weights"].(map[string]any)
	assertConcreteDynamicMap(t, weightsSchema, weigh.RowIDs, true)

	_ = commitWeigh(t, c, lineID, map[string]any{
		loadID + ".1": weighRow(11),
		loadID + ".2": weighRow(22),
	})
	closed, err := c.CloseGroup(t.Context(), "feragame", adminOp(), CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "palletWeight",
	})
	if err != nil {
		t.Fatal(err)
	}
	lineLoad := closed.NextTaskFormBundle
	if lineLoad == nil || lineLoad.TaskID != "lineLoad" || lineLoad.JobInstanceID != lineID {
		t.Fatalf("after weigh close next = %s, want lineLoad@%s", bundleAddress(lineLoad), lineID)
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

func TestHandleLoadJSONOmitsBulkTaskFormBundles(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)

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
	if len(arr) != 0 {
		t.Fatalf("HTTP home Load must omit bulk taskFormBundles: %#v", arr)
	}
	if _, ok := wire["roles"].([]any); !ok {
		t.Fatalf("HTTP Load missing roles: %s", rec.Body.String())
	}
	spawn, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
	if err != nil || spawn == nil {
		t.Fatalf("addressed lots/register after Load: %v", err)
	}
}

func TestCommitNextBundleNotBulk(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	all := FixtureAllRolesOp()
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)
	loaded, err := loader.Load(t.Context(), "feragame", all)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.TaskFormBundles) != 0 {
		t.Fatalf("home Load must omit bulk; got %#v", addresses(loaded.TaskFormBundles))
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "weightCheck", truckID, "", ""); err != nil {
		t.Fatalf("pre-commit addressed weightCheck: %v", err)
	}

	committed, err := c.Commit(t.Context(), "feragame", all, CommitRequest{
		TaskPath:      "intake/weightCheck",
		JobInstanceID: truckID,
		Value:         map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.Next != CommitNextOverview && committed.Next != CommitNextHome && committed.Next != CommitNextTask {
		t.Fatalf("advance Next = %q", committed.Next)
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "weightCheck", truckID, "", ""); err == nil {
		t.Fatal("addressed weightCheck must deny after it leaves available")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "truckArrive", "", "", ""); err != nil {
		t.Fatalf("provisional truckArrive while canStart: %v", err)
	}

	raw, err := json.Marshal(committed)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, has := wire["taskFormBundles"]; has {
		t.Fatalf("Commit JSON must omit taskFormBundles: %s", string(raw))
	}
	if wire["next"] == nil || wire["next"] == "" {
		t.Fatalf("Commit JSON next = %#v", wire["next"])
	}
}

func TestFeragameConfigCommitsWalkAdvanceAndHandoff(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	opLine := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}

	loadID, lineID := seedLineReadyForLineLoad(t, c)

	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", opLine, "intake", "truckArrive", "", "", ""); err == nil {
		t.Fatal("line viewer must not mount intake provisional truckArrive")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", opLine, "line", "openTurno", "", "", ""); err != nil {
		t.Fatalf("line canStart provisional openTurno after first mint: %v", err)
	}

	loaded, err := c.Commit(t.Context(), "feragame", opLine, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NextTaskFormBundle != nil {
		t.Fatalf("line after lineLoad must hand off; got %s", bundleAddress(loaded.NextTaskFormBundle))
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", opLine, "line", "containerReg", lineID, "", ""); err == nil {
		t.Fatal("line viewer must not mount containerReg")
	}
	lineJobWL, ok := worklistJobByID(loaded.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist")
	}
	inst, ok := worklistInstance(lineJobWL, lineID)
	if !ok || inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg status = %#v, want available", inst)
	}
}
