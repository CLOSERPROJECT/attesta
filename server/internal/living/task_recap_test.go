package living

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadTaskRecapFactsWithoutLinks(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	list, err := loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "truckArrive",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %#v", list.Items)
	}
	recap := list.Items[0]
	if recap.UserID != "ada" || recap.Timestamp == "" {
		t.Fatalf("actor/time = %#v", recap)
	}
	if recap.Facts["truckWeight"] != 1000.0 {
		t.Fatalf("facts = %#v", recap.Facts)
	}
	if _, has := recap.Facts["date"]; has {
		t.Fatalf("truckArrive facts must omit date: %#v", recap.Facts)
	}
	if _, has := recap.Facts["turno"]; has {
		t.Fatalf("truckArrive facts must omit turno: %#v", recap.Facts)
	}
	if recap.Titles["truckWeight"] != "Peso lordo camion" {
		t.Fatalf("titles = %#v", recap.Titles)
	}
	if len(recap.Links) != 0 {
		t.Fatalf("truckArrive has no links: %#v", recap.Links)
	}
}

func TestLoadTaskRecapNestedWithStoredLinks(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	links := NewMemoryLinkStore()
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = links
	loader.LinkStore = links

	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())

	var addr SubmissionAddress
	for _, sub := range result.Domain.Submissions {
		if sub.TaskID == "register" {
			addr = SubmissionAddress{
				JobID: sub.JobID, JobInstanceID: sub.JobInstanceID, TaskID: sub.TaskID,
			}
			break
		}
	}
	if addr.TaskID == "" {
		t.Fatalf("register address missing: %#v", result.Domain.Submissions)
	}

	list, err := loader.LoadTaskRecap(t.Context(), "feragame", addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %#v", list.Items)
	}
	recap := list.Items[0]
	if recap.GroupID != "" || recap.GroupInstanceID != "" {
		t.Fatalf("flat register address = %#v", recap)
	}
	if recap.UserID != "ada" || recap.Timestamp == "" {
		t.Fatalf("actor/time = %#v", recap)
	}
	if recap.Facts["client"] != "CO" {
		t.Fatalf("facts = %#v", recap.Facts)
	}
	if recap.Titles["client"] != "Cliente" || recap.Titles["numeroFir"] != "Numero FIR" {
		t.Fatalf("register titles = %#v", recap.Titles)
	}
	if len(recap.Links) != 2 || recap.Links[0].Label != "Stampa etichette" || recap.Links[1].Filename != "reg.csv" {
		t.Fatalf("links = %#v", recap.Links)
	}
}

func TestLoadTaskRecapManySubmitOldestFirstWithPerRowLinks(t *testing.T) {
	clock := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels-1"},
			{URL: "blob:csv-1", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg1.csv"`}},
			{URL: "blob:labels-2"},
			{URL: "blob:csv-2", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg2.csv"`}},
		},
	}
	links := NewMemoryLinkStore()
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = links
	loader.LinkStore = links
	c.Now = func() time.Time {
		now := clock
		clock = clock.Add(time.Second)
		return now
	}

	truckID := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckID, registerLotValue())
	lot2 := registerLotValue()
	lot2["client"] = "DA"
	lot2["numeroFir"] = "FIR-2"
	_ = commitLotRegister(t, c, truckID, lot2)

	addr := SubmissionAddress{JobID: "intake", JobInstanceID: truckID, TaskID: "register"}
	list, err := loader.LoadTaskRecap(t.Context(), "feragame", addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items = %#v", list.Items)
	}
	if list.Items[0].Timestamp >= list.Items[1].Timestamp {
		t.Fatalf("want oldest first: %q then %q", list.Items[0].Timestamp, list.Items[1].Timestamp)
	}
	if list.Items[0].Facts["client"] != "CO" || list.Items[1].Facts["client"] != "DA" {
		t.Fatalf("facts order = %#v %#v", list.Items[0].Facts, list.Items[1].Facts)
	}
	if len(list.Items[0].Links) != 2 || list.Items[0].Links[0].URL != "blob:labels-1" ||
		list.Items[0].Links[1].Filename != "reg1.csv" {
		t.Fatalf("first links = %#v", list.Items[0].Links)
	}
	if len(list.Items[1].Links) != 2 || list.Items[1].Links[0].URL != "blob:labels-2" ||
		list.Items[1].Links[1].Filename != "reg2.csv" {
		t.Fatalf("second links = %#v", list.Items[1].Links)
	}
	if list.Items[0].Titles["client"] != "Cliente" || list.Items[1].Titles["client"] != "Cliente" {
		t.Fatalf("titles = %#v %#v", list.Items[0].Titles, list.Items[1].Titles)
	}
}

func TestLoadTaskRecapRejectsNotDoneAndUnknown(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	_, err := loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "register",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("not-done register = %v, want ErrNotFound", err)
	}

	_, err = loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: "no-such", TaskID: "truckArrive",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown instance = %v, want ErrNotFound", err)
	}

	_, err = loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "truckArrive",
		GroupID: "lots",
	})
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid address err = %v", err)
	}
}

func TestHandleTaskRecapHTTP(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	links := NewMemoryLinkStore()
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = links
	loader.LinkStore = links

	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	h := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return admin, true
		},
	}

	truckID := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckID, registerLotValue())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=register&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("register recap status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire TaskRecapList
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Items) != 1 || wire.Items[0].TaskID != "register" ||
		len(wire.Items[0].Links) != 2 || wire.Items[0].UserID != "ada" {
		t.Fatalf("wire = %#v", wire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("flat recap = %d", rec.Code)
	}
	var onceWire TaskRecapList
	if err := json.Unmarshal(rec.Body.Bytes(), &onceWire); err != nil {
		t.Fatal(err)
	}
	if len(onceWire.Items) != 1 {
		t.Fatalf("once items = %#v", onceWire.Items)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=weightCheck&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not done = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-recap?taskId=truckArrive", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing params = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=register&jobInstanceId="+truckID+"&groupId=lots", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("half group = %d", rec.Code)
	}

	// Load-level: any org Gate identity may read; no Cerbos Commit on this path.
	logH := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}, true
		},
	}
	rec = httptest.NewRecorder()
	logH.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("logistic recap = %d body=%s", rec.Code, rec.Body.String())
	}
}
