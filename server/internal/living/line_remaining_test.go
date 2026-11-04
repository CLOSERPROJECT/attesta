package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func commitLineLoadDone(t *testing.T, c *Committer, lineID, loadID string) {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      GroupTaskPath("line", "lineLoad", ProvisionalGroupInstanceID, "mark"),
		JobInstanceID: lineID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "done",
			},
		},
		Mint: spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func containerRegValue(loadID string, materials map[string][]any) map[string]any {
	v := map[string]any{"sourceLoadId": loadID}
	for k, rows := range materials {
		v[k] = rows
	}
	return v
}

func commitContainerReg(t *testing.T, c *Committer, lineID string, value map[string]any) CommitResult {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      GroupTaskPath("line", "containerReg", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: lineID,
		Value:         value,
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func seedSecondSULoad(t *testing.T, c *Committer) string {
	t.Helper()
	truckID := commitTruckArrive(t, c)
	val := registerLotValue()
	val["client"] = "DA"
	val["palletCount"] = 1.0
	val["truckWeightAfter"] = 650.0
	result := commitLotRegister(t, c, truckID, val)
	for id, facts := range result.Domain.Kind("load") {
		if facts["client"] == "DA" {
			return id
		}
	}
	return firstLoadID(result.Domain)
}

func TestContainerRegUnlocksFromOpenTurnoWithoutLineLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, lineID := seedLineReadyForLineLoad(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	line := lineJob(t)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: lineID, Submissions: domain.Submissions, Tasks: line.Tasks,
		ClosedGroups: domain.ClosedGroups,
	}) {
		t.Fatal("containerReg must unlock from openTurno without lineLoad Submission")
	}
}

func TestProjectWorklistContainerRegInstanceLabelFromLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)
	result := commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 80.0}},
	}))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	wl := ProjectWorklist(mustActiveConfig(t, c), result.Domain, op)
	inst, ok := worklistInstance(worklistMustJob(t, wl, "line"), lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	g, ok := inst.Groups["containerReg"]
	if !ok || len(g.Instances) != 1 {
		t.Fatalf("containerReg group = %#v", g)
	}
	gi := g.Instances[0]
	if gi.Label != loadID {
		t.Fatalf("group instance label = %q, want load id %q", gi.Label, loadID)
	}
	if gi.ID != "containerReg-1" {
		t.Fatalf("group instance id = %q, want containerReg-1", gi.ID)
	}
}

func TestCommitContainerRegMintsContainersAsGroupLeaf(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	result := commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 120.0}},
		"plastica":       {map[string]any{"weight": 40.0}},
	}))
	if result.Domain.Kind("container")["C-1"]["material"] != "Polvere di vetro" ||
		result.Domain.Kind("container")["C-1"]["weight"] != 120.0 ||
		result.Domain.Kind("container")["C-1"]["sourceLoadId"] != loadID {
		t.Fatalf("C-1 = %#v", result.Domain.Kind("container")["C-1"])
	}
	if result.Domain.Kind("container")["C-2"]["material"] != "Plastica" {
		t.Fatalf("C-2 = %#v", result.Domain.Kind("container")["C-2"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "register" || last.GroupID != "containerReg" || last.JobInstanceID != lineID {
		t.Fatalf("submission = %#v", last)
	}
	if last.GroupInstanceID == "" {
		t.Fatal("register must mint a group instance id")
	}
	if !containsString(last.IDsTouched, loadID) {
		t.Fatalf("register must touch load %q for instanceLabelEntity: %#v", loadID, last.IDsTouched)
	}
	if _, has := TaskByID(lineJob(t).Tasks, "loadedExtracted"); has {
		t.Fatal("loadedExtracted must stay off the live line")
	}
}

func TestCommitContainerRegMultiSpawnOnOneJobInstance(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadA, lineID := seedLineReadyForLineLoad(t, c)
	loadB := seedSecondSULoad(t, c)

	_ = commitContainerReg(t, c, lineID, containerRegValue(loadA, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 100.0}},
	}))
	second := commitContainerReg(t, c, lineID, containerRegValue(loadB, map[string][]any{
		"plastica": {map[string]any{"weight": 40.0}},
	}))
	var groupInstances []string
	for _, gi := range second.Domain.GroupInstances {
		if gi.JobID == "line" && gi.JobInstanceID == lineID && gi.GroupID == "containerReg" {
			groupInstances = append(groupInstances, gi.ID)
		}
	}
	if len(groupInstances) != 2 {
		t.Fatalf("containerReg instances = %#v, want 2", groupInstances)
	}
	if len(second.Domain.Kind("container")) < 2 {
		t.Fatalf("containers after multi-spawn = %#v", second.Domain.Kind("container"))
	}
}

func TestCommitContainerRegContinuationDeltaSupersedesPriorIncomplete(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	loadA, lineID := seedLineReadyForLineLoad(t, c)
	loadB := seedSecondSULoad(t, c)
	op := FixtureAllRolesOp()

	first := commitContainerReg(t, c, lineID, containerRegValue(loadA, map[string][]any{
		"polvereDiVetro": {
			map[string]any{"weight": 80.0, "incomplete": true},
		},
		"plastica": {
			map[string]any{"weight": 25.0},
		},
	}))
	prior := first.Domain.Kind("container")["C-1"]
	if prior["incomplete"] != true || prior["weight"] != 80.0 {
		t.Fatalf("prior incomplete = %#v", prior)
	}

	seeded, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "line", "register", lineID, "containerReg", ProvisionalGroupInstanceID)
	if err != nil || seeded == nil {
		t.Fatalf("continuation Spawn Load: %v", err)
	}
	rows, _ := seeded.InitialData["polvereDiVetro"].([]any)
	if len(rows) != 1 {
		t.Fatalf("seeded polvereDiVetro = %#v, want 1 continuation row", seeded.InitialData["polvereDiVetro"])
	}
	seedRow, _ := rows[0].(map[string]any)
	if seedRow["continuesContainerId"] != "C-1" || seedRow["incomplete"] != false || seedRow["weight"] != 80.0 {
		t.Fatalf("continuation seed row = %#v, want prior freeze weight 80", seedRow)
	}
	if _, has := seeded.InitialData["plastica"]; has {
		t.Fatalf("complete plastica must not seed; got %#v", seeded.InitialData["plastica"])
	}
	if _, has := seeded.Schema["properties"].(map[string]any)["secondarySourceLoadId"]; has {
		t.Fatal("secondarySourceLoadId must be absent from schema")
	}

	second := commitContainerReg(t, c, lineID, containerRegValue(loadB, map[string][]any{
		"polvereDiVetro": {
			map[string]any{"weight": 150.0, "continuesContainerId": "C-1", "incomplete": true},
			map[string]any{"weight": 40.0},
		},
	}))
	containers := second.Domain.Kind("container")
	closed := containers["C-1"]
	if closed["incomplete"] != false || closed["weight"] != 80.0 || closed["sourceLoadId"] != loadA {
		t.Fatalf("superseded prior = %#v", closed)
	}
	delta := containers["C-3"]
	if delta["weight"] != 70.0 || delta["sourceLoadId"] != loadB || delta["secondarySourceLoadId"] != loadA {
		t.Fatalf("continuation mint = %#v, want booked 70 secondary %q", delta, loadA)
	}
	if delta["incomplete"] != true {
		t.Fatalf("optional new incomplete = %#v", delta)
	}
	fresh := containers["C-4"]
	if fresh["weight"] != 40.0 || fresh["sourceLoadId"] != loadB {
		t.Fatalf("new row mint = %#v", fresh)
	}
	if _, has := fresh["secondarySourceLoadId"]; has {
		t.Fatalf("new row must omit secondary; got %#v", fresh)
	}
	if _, has := fresh["continuesContainerId"]; has {
		t.Fatalf("continuesContainerId is form-only; got %#v", fresh)
	}
}

func TestRegisterFirstSpawnHasNoContinuationRows(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, lineID := seedLineReadyForLineLoad(t, c)
	op := FixtureAllRolesOp()

	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "line", "register", lineID, "containerReg", ProvisionalGroupInstanceID)
	if err != nil || bundle == nil {
		t.Fatalf("first Spawn Load: %v", err)
	}
	for _, key := range []string{
		"polvereDiVetro", "granellaDiVetro", "granellaDiVetro2", "polvereDiSilicio",
		"plastica", "alluminio", "sopravaglio", "polvereDiAspirazione",
	} {
		if _, has := bundle.InitialData[key]; has {
			t.Fatalf("first Spawn must not seed %s; got %#v", key, bundle.InitialData[key])
		}
	}
}

func TestCommitContainerRegIncompleteFreezeBooksEnteredWeight(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	result := commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {
			map[string]any{"weight": 80.0, "incomplete": true},
			map[string]any{"weight": 55.0},
		},
		"plastica": {
			map[string]any{"weight": 40.0, "incomplete": false},
		},
	}))
	containers := result.Domain.Kind("container")
	open := containers["C-1"]
	if open["incomplete"] != true || open["weight"] != 80.0 || open["sourceLoadId"] != loadID {
		t.Fatalf("incomplete container = %#v", open)
	}
	closedOmit := containers["C-2"]
	if closedOmit["incomplete"] != false || closedOmit["weight"] != 55.0 {
		t.Fatalf("omitted incomplete default = %#v", closedOmit)
	}
	closedExplicit := containers["C-3"]
	if closedExplicit["incomplete"] != false || closedExplicit["weight"] != 40.0 {
		t.Fatalf("explicit incomplete false = %#v", closedExplicit)
	}
}

func TestCommitContainerRegMarksLoadFullyProcessedAndOmitsFromNextSpawn(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	loadA, lineID := seedLineReadyForLineLoad(t, c)
	loadB := seedSecondSULoad(t, c)
	op := FixtureAllRolesOp()

	firstSpawn, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "line", "register", lineID, "containerReg", ProvisionalGroupInstanceID)
	if err != nil || firstSpawn == nil {
		t.Fatalf("first Spawn Load: %v", err)
	}
	firstEnums := loadEnumIDs(t, firstSpawn, "sourceLoadId")
	if !containsString(firstEnums, loadA) || !containsString(firstEnums, loadB) {
		t.Fatalf("first Spawn sourceLoadId enum = %#v, want %q and %q", firstEnums, loadA, loadB)
	}

	result := commitContainerReg(t, c, lineID, containerRegValue(loadA, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 90.0}},
	}))
	if result.Domain.Kind("load")[loadA]["fullyProcessed"] != true {
		t.Fatalf("load %q after register = %#v, want fullyProcessed true", loadA, result.Domain.Kind("load")[loadA])
	}
	if result.Domain.Kind("load")[loadB]["fullyProcessed"] == true {
		t.Fatalf("load %q must stay unset; got %#v", loadB, result.Domain.Kind("load")[loadB])
	}

	secondSpawn, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "line", "register", lineID, "containerReg", ProvisionalGroupInstanceID)
	if err != nil || secondSpawn == nil {
		t.Fatalf("second Spawn Load: %v", err)
	}
	enums := loadEnumIDs(t, secondSpawn, "sourceLoadId")
	if containsString(enums, loadA) {
		t.Fatalf("second Spawn sourceLoadId enum still has fullyProcessed load %q: %#v", loadA, enums)
	}
	if !containsString(enums, loadB) {
		t.Fatalf("second Spawn sourceLoadId enum = %#v, want open load %q", enums, loadB)
	}
	if _, has := secondSpawn.Schema["properties"].(map[string]any)["secondarySourceLoadId"]; has {
		t.Fatal("secondarySourceLoadId must stay off register schema")
	}
}

func loadEnumIDs(t *testing.T, bundle *TaskFormBundle, property string) []string {
	t.Helper()
	props, _ := bundle.Schema["properties"].(map[string]any)
	prop, _ := props[property].(map[string]any)
	raw, ok := prop["enum"]
	if !ok {
		t.Fatalf("%s enum missing: %#v", property, prop)
	}
	switch ids := raw.(type) {
	case []string:
		return ids
	case []any:
		out := make([]string, 0, len(ids))
		for _, v := range ids {
			s, _ := v.(string)
			out = append(out, s)
		}
		return out
	default:
		t.Fatalf("%s enum type = %T", property, raw)
		return nil
	}
}

func TestContainerRegCloseRequiresChildSubmissionNoUnlockSuccessor(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}

	_, err := c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "containerReg",
	})
	var ce *CommitError
	// Committer gates on CanClose (false with zero children) before Domain CloseGroup.
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v, want prep cannot-close", err)
	}

	_ = commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 50.0}},
	}))
	closed, err := c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "containerReg",
	})
	if err != nil {
		t.Fatal(err)
	}
	lineWL, ok := worklistJobByID(closed.Worklist, "line")
	if !ok {
		t.Fatal("missing line")
	}
	wi, ok := worklistInstance(lineWL, lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	if wi.Tasks["containerReg"] != TaskStatusDone {
		t.Fatalf("containerReg after Close = %q, want done (no Unlock successor)", wi.Tasks["containerReg"])
	}
	if closed.Next == CommitNextTask {
		t.Fatalf("Close must not hand off to another Task; next bundle = %s", bundleAddress(closed.NextTaskFormBundle))
	}
}

func TestHandleCommitLineJobEndToEnd(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	roles := []string{"administration", "logistic", "line"}
	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: roles}, true
		},
	}

	post := func(body string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s for %s", rec.Code, rec.Body.String(), body)
		}
		var wire map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
			t.Fatal(err)
		}
		return wire
	}
	postClose := func(jobID, jobInstanceID, groupID string) {
		t.Helper()
		body := `{"jobId":"` + jobID + `","jobInstanceId":"` + jobInstanceID + `","groupId":"` + groupID + `"}`
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/living/close", strings.NewReader(body))
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("close status = %d body=%s", rec.Code, rec.Body.String())
		}
	}

	arrive := post(`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","turno":"1st","truckWeight":1000}}`)
	truckID, _ := arrive["jobInstanceId"].(string)
	reg := post(`{"taskPath":"intake/lots/new/register","jobInstanceId":"` + truckID + `","mint":{"kind":"spawningNew"},"value":{"client":"CO","wasteCode":"160214","numeroFir":"FIR-E2E","palletCount":2,"truckWeightAfter":700,"notes":""}}`)
	postClose("intake", truckID, "lots")

	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(domain)
	_ = reg

	open := post(`{"taskPath":"line/openTurno","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","turno":"1st"}}`)
	lineID, _ := open["jobInstanceId"].(string)
	post(`{"taskPath":"line/palletWeight/new/weigh","jobInstanceId":"` + lineID + `","mint":{"kind":"spawningNew"},"value":{"palletId":"` + loadID + `.1","averagePanelWeight":10,"tareKind":"euro","weight":400,"idoneita":"NIR","panelCount":1}}`)
	post(`{"taskPath":"line/palletWeight/new/weigh","jobInstanceId":"` + lineID + `","mint":{"kind":"spawningNew"},"value":{"palletId":"` + loadID + `.2","averagePanelWeight":10,"tareKind":"euro","weight":450,"idoneita":"NIR","panelCount":1}}`)
	postClose("line", lineID, "palletWeight")
	post(`{"taskPath":"line/lineLoad/new/mark","jobInstanceId":"` + lineID + `","mint":{"kind":"spawningNew"},"value":{"progress":{"` + loadID + `.1":"done","` + loadID + `.2":"done"}}}`)
	postClose("line", lineID, "lineLoad")
	post(`{"taskPath":"line/containerReg/new/register","jobInstanceId":"` + lineID + `","mint":{"kind":"spawningNew"},"value":{"sourceLoadId":"` + loadID + `","polvereDiVetro":[{"weight":120}],"plastica":[{"weight":40}]}}`)
	postClose("line", lineID, "containerReg")

	domain, err = c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Kind("container")) != 2 {
		t.Fatalf("containers = %#v", domain.Kind("container"))
	}
	var sawOpen, sawMark, sawRegister bool
	for _, s := range domain.Submissions {
		if s.JobID != "line" || s.JobInstanceID != lineID {
			continue
		}
		switch s.TaskID {
		case "openTurno":
			sawOpen = true
		case "mark":
			if s.GroupID == "lineLoad" {
				sawMark = true
			}
		case "register":
			if s.GroupID == "containerReg" {
				sawRegister = true
			}
		}
	}
	if !sawOpen || !sawMark || !sawRegister {
		t.Fatalf("line Submissions incomplete: %#v", domain.Submissions)
	}
	if !IsGroupClosed(domain.ClosedGroups, "line", lineID, "containerReg") {
		t.Fatal("containerReg must be closed")
	}
}
