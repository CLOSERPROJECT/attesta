package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func commitLineLoadDone(t *testing.T, c *Committer, shift, intakeID string) {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: shift,
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "done",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFixtureContainerRegUnlockFromInstance(t *testing.T) {
	line := lineJob(t)
	task, ok := line.Tasks.Get("containerReg")
	if !ok {
		t.Fatal("containerReg missing")
	}
	if !taskUnlocksWithInstance(task) {
		t.Fatal("fixture containerReg must set unlockFrom: instance")
	}
	if taskMayBeUnlockGate(lineLoadTask(line)) {
		t.Fatal("lineLoad must stay a non-gate so unlockFrom instance applies")
	}
}

func TestContainerRegUnlocksWithInstanceWithoutLineLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	domain := commitLineShift(t, c, "1st")

	line := lineJob(t)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: "1st", Submissions: domain.Submissions, Tasks: line.Tasks,
	}) {
		t.Fatal("containerReg must unlock with Job instance despite no lineLoad Submission")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "loadedExtracted",
		JobInstanceID: "1st", Submissions: domain.Submissions, Tasks: line.Tasks,
	}) {
		t.Fatal("loadedExtracted stays locked until containerReg Submission")
	}
}

func TestCommitContainerRegMintsContainersAndUnlocksLoadedExtracted(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)
	commitLineShift(t, c, "1st")
	// Authored walk still runs lineLoad; unlock does not require it.
	commitLineLoadDone(t, c, "1st", intakeID)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: "1st",
		Value: map[string]any{
			"sourceLoadId": intakeID,
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
	if result.Domain.Container["C-1"]["material"] != "Polvere di vetro" ||
		result.Domain.Container["C-1"]["weight"] != 120.0 ||
		result.Domain.Container["C-1"]["sourceLoadId"] != intakeID {
		t.Fatalf("C-1 = %#v", result.Domain.Container["C-1"])
	}
	if result.Domain.Container["C-2"]["material"] != "Plastica" {
		t.Fatalf("C-2 = %#v", result.Domain.Container["C-2"])
	}
	if _, has := result.Domain.Container["C-1"]["secondarySourceLoadId"]; has {
		t.Fatal("single-source must omit secondarySourceLoadId")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "containerReg" || last.JobInstanceID != "1st" {
		t.Fatalf("submission = %#v", last)
	}
	rows, _ := last.Facts["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("submission facts rows = %#v", last.Facts)
	}
	if len(last.IDsTouched) != 2 || last.IDsTouched[0] != "C-1" {
		t.Fatalf("idsTouched = %#v", last.IDsTouched)
	}

	line := lineJob(t)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "loadedExtracted",
		JobInstanceID: "1st", Submissions: result.Domain.Submissions, Tasks: line.Tasks,
	}) {
		t.Fatal("containerReg Submission must unlock loadedExtracted")
	}
}

func TestCommitContainerRegDualSourceOmitsSecondaryWeight(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeA := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeA)
	// Second intake load for dual secondary (register only — startsWhen already open).
	opAdmin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration", "logistic"}}
	intakeB := "NIR-261019-SU-1"
	valB := registerValue()
	valB["client"] = "SU"
	valB["palletCount"] = 1.0
	if _, err := c.Commit(t.Context(), "feragame", opAdmin, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: intakeB, Value: valB,
	}); err != nil {
		t.Fatal(err)
	}

	commitLineShift(t, c, "1st")
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: "1st",
		Value: map[string]any{
			"sourceLoadId":          intakeA,
			"secondarySourceLoadId": intakeB,
			"polvereDiVetro": []any{
				map[string]any{"weight": 100.0, "dualSources": true},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c1 := result.Domain.Container["C-1"]
	if c1["secondarySourceLoadId"] != intakeB || c1["sourceLoadId"] != intakeA {
		t.Fatalf("dual container = %#v", c1)
	}
	if _, has := c1["secondaryWeight"]; has {
		t.Fatal("never secondaryWeight")
	}
}

func TestCommitContainerRegWithoutLineLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)
	commitLineShift(t, c, "1st")

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: "1st",
		Value: map[string]any{
			"sourceLoadId": intakeID,
			"alluminio":    []any{map[string]any{"weight": 5.0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Container["C-1"]["material"] != "Alluminio" {
		t.Fatalf("container = %#v", result.Domain.Container)
	}
}

func TestCommitLoadedExtractedRecordsFactsOnly(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)
	commitLineShift(t, c, "1st")
	opLog := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	afterReg, err := c.Commit(t.Context(), "feragame", opLog, CommitRequest{
		TaskPath: "line/containerReg", JobInstanceID: "1st",
		Value: map[string]any{
			"sourceLoadId":   intakeID,
			"polvereDiVetro": []any{map[string]any{"weight": 120.0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	containerCount := len(afterReg.Domain.Container)

	opAdmin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", opAdmin, CommitRequest{
		TaskPath: "line/loadedExtracted", JobInstanceID: "1st",
		Value: map[string]any{
			"checkedLoadedWeight": 850.0,
			"checkedContainerSum": 130.0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Domain.Container) != containerCount {
		t.Fatal("loadedExtracted must not mint or patch containers")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "loadedExtracted" {
		t.Fatalf("submission = %#v", last)
	}
	if last.Facts["checkedLoadedWeight"] != 850.0 || last.Facts["checkedContainerSum"] != 130.0 {
		t.Fatalf("facts = %#v", last.Facts)
	}
	if len(last.IDsTouched) != 0 {
		t.Fatalf("idsTouched = %#v", last.IDsTouched)
	}
}

func TestCommitLoadedExtractedDeniedViaCerbosBeforeContainerReg(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	before := commitLineShift(t, c, "1st")

	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	if sequenceOkForCommit(cfg, before, "line", "loadedExtracted", "1st") {
		t.Fatal("sequenceOk must be false before containerReg")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/loadedExtracted", JobInstanceID: "1st",
		Value: map[string]any{
			"checkedLoadedWeight": 850.0,
			"checkedContainerSum": 120.0,
		},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Submissions) != len(before.Submissions) {
		t.Fatal("locked loadedExtracted must not append Submission")
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

	post := func(body string) CommitResult {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s for %s", rec.Code, rec.Body.String(), body)
		}
		var got CommitResult
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	intakeID := "NIR-261019-NO-1"
	post(`{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":900,"truckTare":50,"palletCount":2,"notes":""}}`)
	post(`{"taskPath":"intake/codiceFir","jobInstanceId":"NIR-261019-NO-1","value":{"codiceFir":"FIR-E2E"}}`)
	post(`{"taskPath":"intake/palletWeight","jobInstanceId":"NIR-261019-NO-1","value":{"averagePanelWeight":10,"tareKind":"euro","weights":{"NIR-261019-NO-1.1":{"weight":400,"tareKind":"euro","idoneita":"NIR","panelCount":1},"NIR-261019-NO-1.2":{"weight":450,"tareKind":"euro","idoneita":"NIR","panelCount":1}}}}`)
	post(`{"taskPath":"line/shift","jobInstanceId":"1st","value":{"shift":"1st"}}`)
	post(`{"taskPath":"line/lineLoad","jobInstanceId":"1st","value":{"progress":{"NIR-261019-NO-1.1":"done","NIR-261019-NO-1.2":"done"}}}`)
	afterContainers := post(`{"taskPath":"line/containerReg","jobInstanceId":"1st","value":{"sourceLoadId":"NIR-261019-NO-1","polvereDiVetro":[{"weight":120}],"plastica":[{"weight":40}]}}`)
	if len(afterContainers.Domain.Container) != 2 {
		t.Fatalf("containers = %#v", afterContainers.Domain.Container)
	}
	final := post(`{"taskPath":"line/loadedExtracted","jobInstanceId":"1st","value":{"checkedLoadedWeight":850,"checkedContainerSum":160}}`)

	var sawShift, sawLineLoad, sawContainerReg, sawLoadedExtracted bool
	for _, s := range final.Domain.Submissions {
		if s.JobID != "line" || s.JobInstanceID != "1st" {
			continue
		}
		switch s.TaskID {
		case "shift":
			sawShift = true
		case "lineLoad":
			sawLineLoad = true
		case "containerReg":
			sawContainerReg = true
		case "loadedExtracted":
			sawLoadedExtracted = true
		}
	}
	if !sawShift || !sawLineLoad || !sawContainerReg || !sawLoadedExtracted {
		t.Fatalf("line Submissions incomplete: %#v", final.Domain.Submissions)
	}
	if final.Domain.Container["C-1"]["sourceLoadId"] != intakeID {
		t.Fatalf("C-1 = %#v", final.Domain.Container["C-1"])
	}

	loadRec := httptest.NewRecorder()
	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d", loadRec.Code)
	}
	var loaded LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Domain.Container) != 2 {
		t.Fatalf("load containers = %#v", loaded.Domain.Container)
	}
	last := loaded.Domain.Submissions[len(loaded.Domain.Submissions)-1]
	if last.TaskID != "loadedExtracted" {
		t.Fatalf("last submission = %#v", last)
	}
}
