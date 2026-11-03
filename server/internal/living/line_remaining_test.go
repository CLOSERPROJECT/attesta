package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func commitLineLoadDone(t *testing.T, c *Committer, lineID, loadID string) {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "done",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestContainerRegLockedUntilLineLoadSubmission(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, lineID := seedLineReadyForLineLoad(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	line := lineJob(t)
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: lineID, Submissions: domain.Submissions, Tasks: line.Tasks,
		ClosedGroups: domain.ClosedGroups,
	}) {
		t.Fatal("containerReg must stay locked until lineLoad Submission")
	}
}

func TestCommitContainerRegMintsContainersAndUnlocksLoadedExtracted(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)
	commitLineLoadDone(t, c, lineID, loadID)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: lineID,
		Value: map[string]any{
			"sourceLoadId": loadID,
			"polvereDiVetro": []any{
				map[string]any{"weight": 120.0},
			},
			"plastica": []any{
				map[string]any{"weight": 40.0},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("container")["C-1"]["material"] != "Polvere di vetro" ||
		result.Domain.Kind("container")["C-1"]["weight"] != 120.0 ||
		result.Domain.Kind("container")["C-1"]["sourceLoadId"] != loadID {
		t.Fatalf("C-1 = %#v", result.Domain.Kind("container")["C-1"])
	}
	if result.Domain.Kind("container")["C-2"]["material"] != "Plastica" {
		t.Fatalf("C-2 = %#v", result.Domain.Kind("container")["C-2"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "containerReg" || last.JobInstanceID != lineID {
		t.Fatalf("submission = %#v", last)
	}

	line := lineJob(t)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "loadedExtracted",
		JobInstanceID: lineID, Submissions: result.Domain.Submissions, Tasks: line.Tasks,
		ClosedGroups: result.Domain.ClosedGroups,
	}) {
		t.Fatal("containerReg Submission must unlock loadedExtracted")
	}
}

func TestCommitContainerRegDualSourceOmitsSecondaryWeight(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadA, lineID := seedLineReadyForLineLoad(t, c)
	commitLineLoadDone(t, c, lineID, loadA)

	truckID := commitTruckArrive(t, c)
	valB := registerLotValue()
	valB["client"] = "SU"
	valB["palletCount"] = 1.0
	valB["truckWeightAfter"] = 650.0
	resultB := commitLotRegister(t, c, truckID, valB)
	loadB := firstLoadID(resultB.Domain)
	// Prefer the newly minted SU load (firstLoadID may return NO from prior truck).
	for id, facts := range resultB.Domain.Kind("load") {
		if facts["client"] == "SU" {
			loadB = id
			break
		}
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: lineID,
		Value: map[string]any{
			"sourceLoadId":          loadA,
			"secondarySourceLoadId": loadB,
			"polvereDiVetro": []any{
				map[string]any{"weight": 100.0, "dualSources": true},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c1 := result.Domain.Kind("container")["C-1"]
	if c1["secondarySourceLoadId"] != loadB || c1["sourceLoadId"] != loadA {
		t.Fatalf("dual container = %#v", c1)
	}
	if _, has := c1["secondaryWeight"]; has {
		t.Fatal("never secondaryWeight")
	}
}

func TestCommitLoadedExtractedRecordsFactsOnly(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)
	commitLineLoadDone(t, c, lineID, loadID)
	opLog := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	afterReg, err := c.Commit(t.Context(), "feragame", opLog, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: lineID,
		Value: map[string]any{
			"sourceLoadId":   loadID,
			"polvereDiVetro": []any{map[string]any{"weight": 120.0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	containerCount := len(afterReg.Domain.Kind("container"))

	opAdmin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", opAdmin, CommitRequest{
		TaskPath: "line/loadedExtracted", JobInstanceID: lineID,
		Value: map[string]any{
			"checkedLoadedWeight": 850.0,
			"checkedContainerSum": 130.0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Domain.Kind("container")) != containerCount {
		t.Fatal("loadedExtracted must not mint or patch containers")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "loadedExtracted" {
		t.Fatalf("submission = %#v", last)
	}
	if last.Facts["checkedLoadedWeight"] != 850.0 || last.Facts["checkedContainerSum"] != 130.0 {
		t.Fatalf("facts = %#v", last.Facts)
	}
}

func TestCommitLoadedExtractedDeniedViaCerbosBeforeContainerReg(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	_, lineID := seedLineReadyForLineLoad(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if sequenceOkForCommit(cfg, before, "line", "loadedExtracted", lineID) {
		t.Fatal("sequenceOk must be false before containerReg")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/loadedExtracted", JobInstanceID: lineID,
		Value: map[string]any{
			"checkedLoadedWeight": 850.0,
			"checkedContainerSum": 120.0,
		},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
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
	reg := post(`{"taskPath":"intake/lots/new/register","jobInstanceId":"` + truckID + `","mint":{"kind":"spawningNew"},"value":{"client":"NO","wasteCode":"191212","numeroFir":"FIR-E2E","palletCount":2,"truckWeightAfter":700,"notes":""}}`)
	postClose("intake", truckID, "lots")

	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(domain)
	_ = reg

	open := post(`{"taskPath":"line/openTurno","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","turno":"1st"}}`)
	lineID, _ := open["jobInstanceId"].(string)
	post(`{"taskPath":"line/palletWeight/new/weigh","jobInstanceId":"` + lineID + `","mint":{"kind":"spawningNew"},"value":{"averagePanelWeight":10,"tareKind":"euro","weights":{"` + loadID + `.1":{"weight":400,"tareKind":"euro","idoneita":"NIR","panelCount":1},"` + loadID + `.2":{"weight":450,"tareKind":"euro","idoneita":"NIR","panelCount":1}}}}`)
	postClose("line", lineID, "palletWeight")
	post(`{"taskPath":"line/lineLoad","jobInstanceId":"` + lineID + `","value":{"progress":{"` + loadID + `.1":"done","` + loadID + `.2":"done"}}}`)
	post(`{"taskPath":"line/containerReg","jobInstanceId":"` + lineID + `","value":{"sourceLoadId":"` + loadID + `","polvereDiVetro":[{"weight":120}],"plastica":[{"weight":40}]}}`)
	post(`{"taskPath":"line/loadedExtracted","jobInstanceId":"` + lineID + `","value":{"checkedLoadedWeight":850,"checkedContainerSum":160}}`)

	domain, err = c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Kind("container")) != 2 {
		t.Fatalf("containers = %#v", domain.Kind("container"))
	}
	var sawOpen, sawLineLoad, sawContainerReg, sawLoadedExtracted bool
	for _, s := range domain.Submissions {
		if s.JobID != "line" || s.JobInstanceID != lineID {
			continue
		}
		switch s.TaskID {
		case "openTurno":
			sawOpen = true
		case "lineLoad":
			sawLineLoad = true
		case "containerReg":
			sawContainerReg = true
		case "loadedExtracted":
			sawLoadedExtracted = true
		}
	}
	if !sawOpen || !sawLineLoad || !sawContainerReg || !sawLoadedExtracted {
		t.Fatalf("line Submissions incomplete: %#v", domain.Submissions)
	}
}
