package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	ensureOpenWorkingDay(t, c)
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
	wd, ok := worklistJobByID(result.Worklist, "workingDay")
	if !ok || !wd.CanStart {
		t.Fatalf("workingDay canStart = %#v", wd)
	}
	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok || intake.CanStart {
		t.Fatalf("intake canStart = %#v, want false without open day", intake)
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

	provisional, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "register", truckID, "", "")
	if err != nil || provisional == nil {
		t.Fatalf("addressed register: %v %#v", err, provisional)
	}
	// Prior truck still incomplete → Nuovo camion / provisional truckArrive off.
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", all, "intake", "truckArrive", "", "", ""); err == nil {
		t.Fatal("provisional truckArrive must deny while prior truck incomplete")
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

func TestLoadTaskBundlePalletWeightFlatForm(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)

	home, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	if len(home.TaskFormBundles) != 0 {
		t.Fatalf("home Load must omit bulk; got %#v", addresses(home.TaskFormBundles))
	}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", FixtureAllRolesOp(), "line", "weigh", lineID, "", "")
	if err != nil || bundle == nil {
		t.Fatalf("missing weigh: %v", err)
	}
	if bundle.Surface != "live" {
		t.Fatalf("surface = %q", bundle.Surface)
	}
	if len(bundle.RowIDs) != 0 {
		t.Fatalf("flat weigh must not project map rowIds; got %#v", bundle.RowIDs)
	}
	if bundle.ProjectionKey != "" {
		t.Fatalf("flat weigh projectionKey must be empty; got %q", bundle.ProjectionKey)
	}
	if rule, ok := bundle.Derived["panelCount"]; !ok || rule.Expr == "" {
		t.Fatalf("derived panelCount = %#v", bundle.Derived)
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
	if load, ok := domainBound["lot"].(map[string]any); !ok || len(load) == 0 {
		t.Fatalf("bindings.domain.lot = %#v, want minted lot kind", domainBound["lot"])
	}
	subs, ok := domainBound["submissions"].([]any)
	if !ok || len(subs) == 0 {
		t.Fatalf("bindings.domain.submissions = %#v", domainBound["submissions"])
	}
	if _, has := bundle.InitialData["weights"]; has {
		t.Fatalf("flat weigh must not seed weights map; got %#v", bundle.InitialData)
	}
	if bundle.InitialData["averagePanelWeight"] != 20 {
		t.Fatalf("averagePanelWeight default = %#v, want 20", bundle.InitialData["averagePanelWeight"])
	}
	if bundle.InitialData["tareKind"] != "industrial" {
		t.Fatalf("tareKind default = %#v, want industrial", bundle.InitialData["tareKind"])
	}
	if bundle.InitialData["idoneita"] != "NIR" {
		t.Fatalf("idoneita default = %#v, want NIR", bundle.InitialData["idoneita"])
	}
	props, _ := bundle.Schema["properties"].(map[string]any)
	if _, has := props["weights"]; has {
		t.Fatalf("flat weigh schema must not include weights; got %#v", props)
	}
	palletProp, _ := props["palletId"].(map[string]any)
	enumIDs, _ := palletProp["enum"].([]string)
	found := false
	for _, id := range enumIDs {
		if id == loadID+".1" || id == loadID+".2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("palletId enum = %#v, want unweighted pallets for %s", enumIDs, loadID)
	}
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

func TestMarkProgressPartialMapAfterWeigh(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	mark, err := BuildTaskFormBundle(mustActiveConfig(t, c), mustDomain(t, c), ActionableTaskRef{
		JobID: "line", TaskID: "markProgress", JobInstanceID: lineID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mark.TaskID != "markProgress" || mark.GroupID != "" || mark.JobInstanceID != lineID {
		t.Fatalf("leaf form = %s, want markProgress@%s", bundleAddress(mark), lineID)
	}
	if len(mark.RowIDs) == 0 {
		t.Fatal("markProgress partial-map still projects Domain-read row ids")
	}
	progressProps, _ := mark.Schema["properties"].(map[string]any)
	progressSchema, _ := progressProps["progress"].(map[string]any)
	assertConcreteDynamicMap(t, progressSchema, mark.RowIDs, false)
	assertLineLoadCellMapBundle(t, mark)
	_ = loadID
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
	spawn, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "intake", "register", truckID, "", "")
	if err != nil || spawn == nil {
		t.Fatalf("addressed register after Load: %v", err)
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
	if committed.Navigation != CommitNextHome && committed.Navigation != CommitNextTask {
		t.Fatalf("advance Navigation = %q", committed.Navigation)
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
	if wire["navigation"] == nil || wire["navigation"] == "" {
		t.Fatalf("Commit JSON navigation = %#v", wire["navigation"])
	}
	if _, has := wire["next"]; has {
		t.Fatalf("Commit JSON must not ship legacy next: %s", string(raw))
	}
}

func TestFeragameConfigCommitsWalkAdvanceAndHandoff(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	opLine := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}

	loadID, lineID := seedLineReadyForLineLoad(t, c)

	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", opLine, "intake", "truckArrive", "", "", ""); err == nil {
		t.Fatal("line viewer must not mount intake provisional truckArrive")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", opLine, "line", "openTurno", "", "", ""); err == nil {
		t.Fatal("line must not canStart openTurno while a shift is still open")
	}

	loaded, err := c.Commit(t.Context(), "feragame", opLine, CommitRequest{
		TaskPath:      "line/markProgress",
		JobInstanceID: lineID,
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
	if loaded.Navigation != CommitNextStay {
		t.Fatalf("after markProgress Navigation = %q, want stay", loaded.Navigation)
	}
	if loaded.NextTaskFormBundle == nil || loaded.NextTaskFormBundle.TaskID != "markProgress" {
		t.Fatalf("stay bundle = %#v, want markProgress", loaded.NextTaskFormBundle)
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
		t.Fatalf("containerReg status = %#v, want available (logistic-only)", inst)
	}
	if inst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress = %q, want actionable (many + leftover rows)", inst.Tasks["markProgress"])
	}
}

func flatSchemaDefaultsProbeConfig() OrgConfig {
	return OrgConfig{
		ID:      "probe",
		Version: "1",
		Jobs: OrderedObject{
			Keys: []string{"line"},
			Values: map[string]any{
				"line": Job{
					Repeatable: true,
					InstanceID: `"x"`,
					Label:      `"x"`,
					Tasks: []TaskConfig{
						{
							"id":    "weigh",
							"roles": []any{"logistic"},
							"schema": map[string]any{
								"type":     "object",
								"title":    "Pesa bancale",
								"required": []any{"palletId", "averagePanelWeight", "tareKind", "weight", "idoneita"},
								"properties": map[string]any{
									"palletId":           map[string]any{"type": "string", "title": "Bancale"},
									"averagePanelWeight": map[string]any{"type": "number", "exclusiveMinimum": 0, "title": "Peso medio", "default": 20},
									"tareKind": map[string]any{
										"type": "string", "title": "Tara",
										"enum": []any{"euro", "industrial", "custom"}, "default": "euro",
									},
									"customTareWeight": map[string]any{"type": "number", "exclusiveMinimum": 0},
									"weight":           map[string]any{"type": "number", "exclusiveMinimum": 0},
									"idoneita": map[string]any{
										"type": "string", "title": "IR / NIR",
										"enum": []any{"NIR", "IR"}, "default": "NIR",
									},
									"panelCount": map[string]any{"type": "number"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestBuildTaskFormBundleFlatSeedsSchemaDefaults(t *testing.T) {
	bundle, err := BuildTaskFormBundle(flatSchemaDefaultsProbeConfig(), EmptyDomain(), ActionableTaskRef{
		JobID: "line", TaskID: "weigh",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := bundle.InitialData
	if got["averagePanelWeight"] != 20 {
		t.Fatalf("averagePanelWeight = %#v, want 20", got["averagePanelWeight"])
	}
	if got["tareKind"] != "euro" {
		t.Fatalf("tareKind = %#v, want euro", got["tareKind"])
	}
	if got["idoneita"] != "NIR" {
		t.Fatalf("idoneita = %#v, want NIR", got["idoneita"])
	}
	for _, key := range []string{"palletId", "weight", "customTareWeight", "panelCount"} {
		if _, has := got[key]; has {
			t.Fatalf("must not invent %q without default; got %#v", key, got)
		}
	}
}

func TestBuildTaskFormBundleFlatRequiredEnumFallsBackToFirst(t *testing.T) {
	cfg := OrgConfig{
		ID:      "probe",
		Version: "1",
		Jobs: OrderedObject{
			Keys: []string{"job"},
			Values: map[string]any{
				"job": Job{
					Repeatable: true,
					InstanceID: `"x"`,
					Label:      `"x"`,
					Tasks: []TaskConfig{
						{
							"id":    "pick",
							"roles": []any{"logistic"},
							"schema": map[string]any{
								"type":     "object",
								"required": []any{"status"},
								"properties": map[string]any{
									"status": map[string]any{
										"type": "string",
										"enum": []any{"open", "closed"},
									},
									"note": map[string]any{
										"type": "string",
										"enum": []any{"a", "b"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{
		JobID: "job", TaskID: "pick",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["status"] != "open" {
		t.Fatalf("required enum without default = %#v, want first enum open", bundle.InitialData)
	}
	if _, has := bundle.InitialData["note"]; has {
		t.Fatalf("optional enum without default must stay unset; got %#v", bundle.InitialData)
	}
}

func TestBuildTaskFormBundleObjectMapKeepsRowAndFormDefaults(t *testing.T) {
	cfg := OrgConfig{
		ID:      "probe",
		Version: "1",
		Domain: map[string]any{
			"entities": map[string]any{
				"pallet": map[string]any{},
			},
		},
		Jobs: OrderedObject{
			Keys: []string{"line"},
			Values: map[string]any{
				"line": Job{
					Repeatable: true,
					InstanceID: `"x"`,
					Label:      `"x"`,
					Tasks: []TaskConfig{
						{
							"id":    "weigh",
							"roles": []any{"logistic"},
							"dynamicProperties": map[string]any{
								"weights": map[string]any{"read": "rows"},
							},
							"reads": map[string]any{
								"rows": map[string]any{"entity": "pallet", "key": "id"},
							},
							"schema": map[string]any{
								"type":     "object",
								"required": []any{"averagePanelWeight", "tareKind", "weights"},
								"properties": map[string]any{
									"averagePanelWeight": map[string]any{"type": "number", "default": 20},
									"tareKind": map[string]any{
										"type": "string", "enum": []any{"euro", "industrial"}, "default": "euro",
									},
									"weights": map[string]any{
										"type": "object",
										"additionalProperties": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"tareKind": map[string]any{
													"type": "string", "enum": []any{"euro", "industrial"}, "default": "euro",
												},
												"idoneita": map[string]any{
													"type": "string", "enum": []any{"NIR", "IR"},
												},
												"weight": map[string]any{"type": "number"},
											},
										},
									},
								},
							},
							"uiSchema": map[string]any{
								"weights": map[string]any{"ui:field": "mapTable"},
							},
						},
					},
				},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"pallet": {"P1": {}, "P2": {}},
	})
	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "line", TaskID: "weigh", JobInstanceID: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["averagePanelWeight"] != 20 || bundle.InitialData["tareKind"] != "euro" {
		t.Fatalf("form defaults = %#v", bundle.InitialData)
	}
	weights, _ := bundle.InitialData["weights"].(map[string]any)
	if len(weights) != 2 {
		t.Fatalf("row seed = %#v, want both rows", weights)
	}
	for _, id := range []string{"P1", "P2"} {
		row, _ := weights[id].(map[string]any)
		if row["tareKind"] != "euro" {
			t.Fatalf("row %s = %#v, want authored tareKind default only", id, row)
		}
		if _, has := row["idoneita"]; has {
			t.Fatalf("row %s must not invent first-enum idoneita; got %#v", id, row)
		}
	}
}

// Fail closed: form seed stays generic — no plant Task id branches.
func TestFormBundleSourceOmitsPlantTaskIDs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "form_bundle.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	for _, token := range []string{`"lineLoad"`, `"feragame"`, `"openTurno"`, `"palletWeight"`, `"containerReg"`} {
		if strings.Contains(src, token) {
			t.Fatalf("form_bundle.go must not hard-code plant token %s", token)
		}
	}
}

func authoredInitialDataProbeConfig(initialData any) OrgConfig {
	task := TaskConfig{
		"id":    "seedProbe",
		"roles": []any{"logistic"},
		"schema": map[string]any{
			"type":  "object",
			"title": "Seed probe",
			"properties": map[string]any{
				"note": map[string]any{"type": "string", "default": "schema-default"},
				"tag":  map[string]any{"type": "string"},
				"meta": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"source": map[string]any{"type": "string"},
						"open":   map[string]any{"type": "boolean"},
					},
				},
				"rows": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"continuesId": map[string]any{"type": "string"},
							"label":       map[string]any{"type": "string"},
							"flag":        map[string]any{"type": "boolean", "default": true},
						},
					},
				},
			},
		},
		"onSubmit": map[string]any{
			"transform": []any{
				map[string]any{
					"op":     "mint",
					"entity": "item",
					"id":     `"I1"`,
					"set": map[string]any{
						"note": "$.note",
						"tag":  "$.tag",
					},
				},
			},
		},
	}
	if initialData != nil {
		task["initialData"] = initialData
	}
	return OrgConfig{
		ID:      "seed-probe",
		Org:     "seedorg",
		Version: "1",
		Domain: map[string]any{
			"entities": map[string]any{
				"item": map[string]any{},
			},
		},
		Catalogs: map[string]any{
			"materials": []any{
				map[string]any{"key": "glass", "label": "Glass"},
			},
		},
		Jobs: OrderedObject{
			Keys: []string{"probe"},
			Values: map[string]any{
				"probe": Job{
					Repeatable: true,
					StartLabel: "Start probe",
					InstanceID: `"P1"`,
					Label:      `"probe"`,
					Tasks:      []TaskConfig{task},
				},
			},
		},
	}
}

func TestBuildTaskFormBundleAuthoredInitialDataFromDomainAndCatalogs(t *testing.T) {
	cfg := authoredInitialDataProbeConfig(`(
  $open := $sift($domain.item, function($v){$v.open = true});
  {
    "tag": $catalogs.materials[0].key,
    "rows": $map($keys($open), function($id){(
      $facts := $lookup($open, $id);
      {"continuesId": $id, "label": $facts.label, "flag": false}
    )})
  }
)`)
	domain := domainFromKinds(map[string]EntityMap{
		"item": {
			"A": {"open": true, "label": "alpha"},
			"B": {"open": false, "label": "beta"},
			"C": {"open": true, "label": "gamma"},
		},
	})
	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "probe", TaskID: "seedProbe", JobInstanceID: "P1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["tag"] != "glass" {
		t.Fatalf("catalog seed tag = %#v", bundle.InitialData["tag"])
	}
	if bundle.InitialData["note"] != "schema-default" {
		t.Fatalf("schema default must fill gaps; note = %#v", bundle.InitialData["note"])
	}
	rows, _ := bundle.InitialData["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows = %#v, want 2 open items", bundle.InitialData["rows"])
	}
	got := map[string]map[string]any{}
	for _, raw := range rows {
		row, _ := raw.(map[string]any)
		id, _ := row["continuesId"].(string)
		got[id] = row
	}
	for _, id := range []string{"A", "C"} {
		row := got[id]
		if row == nil || row["flag"] != false {
			t.Fatalf("row %s = %#v, want flag false", id, row)
		}
	}
	if _, has := got["B"]; has {
		t.Fatalf("closed item B must not seed; got %#v", got)
	}
}

func TestBuildTaskFormBundleOmitsBlankInitialDataExpr(t *testing.T) {
	for _, expr := range []any{nil, "", "   "} {
		cfg := authoredInitialDataProbeConfig(expr)
		bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{
			JobID: "probe", TaskID: "seedProbe",
		})
		if err != nil {
			t.Fatalf("expr=%#v: %v", expr, err)
		}
		if bundle.InitialData["note"] != "schema-default" {
			t.Fatalf("expr=%#v note = %#v", expr, bundle.InitialData["note"])
		}
		if _, has := bundle.InitialData["rows"]; has {
			t.Fatalf("expr=%#v must not invent rows; got %#v", expr, bundle.InitialData)
		}
		if _, has := bundle.InitialData["meta"]; has {
			t.Fatalf("expr=%#v must not invent meta; got %#v", expr, bundle.InitialData)
		}
	}
}

func TestBuildTaskFormBundleInitialDataRejectsNonObject(t *testing.T) {
	cfg := authoredInitialDataProbeConfig(`["not", "an", "object"]`)
	_, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{
		JobID: "probe", TaskID: "seedProbe",
	})
	if err == nil {
		t.Fatal("expected initialData non-object error")
	}
}

func TestLoadTaskBundleAuthoredInitialDataEndToEnd(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := authoredInitialDataProbeConfig(`(
  $facts := $lookup($domain.item, "X1");
  {
    "tag": $catalogs.materials[0].key,
    "rows": [{"continuesId": "X1", "label": $facts.label, "flag": false}],
    "meta": {"source": $facts.label, "open": $facts.open}
  }
)`)
	if err := configs.Put(t.Context(), "seedorg", "1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "seedorg", "1"); err != nil {
		t.Fatal(err)
	}
	domain := domainFromKinds(map[string]EntityMap{
		"item": {
			"X1": {"label": "from-domain", "open": true},
		},
	})
	if err := domains.Put(t.Context(), "seedorg", domain); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	op := OperatorIdentity{UserID: "bo", Org: "seedorg", Roles: []string{"logistic"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "seedorg", op, "probe", "seedProbe", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["tag"] != "glass" {
		t.Fatalf("Load seed tag = %#v", bundle.InitialData["tag"])
	}
	rows, _ := bundle.InitialData["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("Load seed rows = %#v", bundle.InitialData["rows"])
	}
	row, _ := rows[0].(map[string]any)
	if row["continuesId"] != "X1" || row["flag"] != false {
		t.Fatalf("Load seed row = %#v", row)
	}
	meta, _ := bundle.InitialData["meta"].(map[string]any)
	if meta["source"] != "from-domain" || meta["open"] != true {
		t.Fatalf("Load seed object meta = %#v", bundle.InitialData["meta"])
	}
	if bundle.InitialData["note"] != "schema-default" {
		t.Fatalf("Load schema default note = %#v", bundle.InitialData["note"])
	}
}

// Authored initialData is Load-only: Commit still applies client Value on the same Task.
func TestCommitAuthoredInitialDataTaskUsesClientValue(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := authoredInitialDataProbeConfig(`{
  "tag": "seed-tag",
  "note": "seed-note",
  "meta": {"source": "seed", "open": true}
}`)
	if err := configs.Put(t.Context(), "seedorg", "1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "seedorg", "1"); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), "seedorg", EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	op := OperatorIdentity{UserID: "bo", Org: "seedorg", Roles: []string{"logistic"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "seedorg", op, "probe", "seedProbe", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["tag"] != "seed-tag" || bundle.InitialData["note"] != "seed-note" {
		t.Fatalf("Load seed = %#v", bundle.InitialData)
	}

	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
	}
	result, err := c.Commit(t.Context(), "seedorg", op, CommitRequest{
		TaskPath: "probe/seedProbe",
		Value:    map[string]any{"note": "from-client", "tag": "client-tag"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	item := result.Domain.Kind("item")["I1"]
	if item["note"] != "from-client" || item["tag"] != "client-tag" {
		t.Fatalf("Commit must use client Value, not Load seed; item = %#v", item)
	}
}

func TestBuildTaskFormBundleCellMapDoesNotInventCells(t *testing.T) {
	cfg := OrgConfig{
		ID:      "probe",
		Version: "1",
		Domain: map[string]any{
			"entities": map[string]any{
				"pallet": map[string]any{},
			},
		},
		Jobs: OrderedObject{
			Keys: []string{"line"},
			Values: map[string]any{
				"line": Job{
					Repeatable: true,
					InstanceID: `"x"`,
					Label:      `"x"`,
					Tasks: []TaskConfig{
						{
							"id":    "lineLoad",
							"roles": []any{"line"},
							"dynamicProperties": map[string]any{
								"progress": map[string]any{"read": "rows"},
							},
							"reads": map[string]any{
								"rows": map[string]any{"entity": "pallet", "key": "id"},
							},
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"note": map[string]any{"type": "string", "default": "ready"},
									"progress": map[string]any{
										"type": "object",
										"additionalProperties": map[string]any{
											"type": "string",
											"enum": []any{"partial", "done"},
										},
									},
								},
							},
							"uiSchema": map[string]any{
								"progress": map[string]any{"ui:field": "mapTable"},
							},
						},
					},
				},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"pallet": {"P1": {}, "P2": {"progress": "partial"}},
	})
	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "line", TaskID: "lineLoad", JobInstanceID: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.InitialData["note"] != "ready" {
		t.Fatalf("cell-map form default = %#v", bundle.InitialData["note"])
	}
	progress, _ := bundle.InitialData["progress"].(map[string]any)
	if len(progress) != 1 || progress["P2"] != "partial" {
		t.Fatalf("cell seed = %#v, want only existing P2=partial", progress)
	}
}
