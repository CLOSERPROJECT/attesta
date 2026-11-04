package living

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadTaskRecapFactsWithoutLinks(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	recap, err := loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "truckArrive",
	})
	if err != nil {
		t.Fatal(err)
	}
	if recap.UserID != "ada" || recap.Timestamp == "" {
		t.Fatalf("actor/time = %#v", recap)
	}
	if recap.Facts["date"] != "2026-10-19" || recap.Facts["turno"] != "1st" {
		t.Fatalf("facts = %#v", recap.Facts)
	}
	if recap.Titles["date"] != "Data" ||
		recap.Titles["turno"] != "Turno" ||
		recap.Titles["truckWeight"] != "Peso lordo camion" {
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
				GroupID: sub.GroupID, GroupInstanceID: sub.GroupInstanceID,
			}
			break
		}
	}
	if addr.GroupInstanceID == "" {
		t.Fatalf("register address missing: %#v", result.Domain.Submissions)
	}

	recap, err := loader.LoadTaskRecap(t.Context(), "feragame", addr)
	if err != nil {
		t.Fatal(err)
	}
	if recap.GroupID != "lots" || recap.GroupInstanceID != addr.GroupInstanceID {
		t.Fatalf("nested address = %#v", recap)
	}
	if recap.UserID != "ada" || recap.Timestamp == "" {
		t.Fatalf("actor/time = %#v", recap)
	}
	if recap.Facts["client"] != "CO" {
		t.Fatalf("facts = %#v", recap.Facts)
	}
	if recap.Titles["client"] != "Cliente" || recap.Titles["numeroFir"] != "Numero FIR" {
		t.Fatalf("nested titles = %#v", recap.Titles)
	}
	if len(recap.Links) != 2 || recap.Links[0].Label != "Stampa etichette" || recap.Links[1].Filename != "reg.csv" {
		t.Fatalf("links = %#v", recap.Links)
	}
}

func TestLoadTaskRecapRejectsNotDoneAndUnknown(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	_, err := loader.LoadTaskRecap(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "register",
		GroupID: "lots", GroupInstanceID: "ghost",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("not-done nested = %v, want ErrNotFound", err)
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
	result := commitLotRegister(t, c, truckID, registerLotValue())
	var groupInstanceID string
	for _, sub := range result.Domain.Submissions {
		if sub.TaskID == "register" {
			groupInstanceID = sub.GroupInstanceID
			break
		}
	}
	if groupInstanceID == "" {
		t.Fatal("missing group instance")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=register&jobInstanceId="+truckID+
			"&groupId=lots&groupInstanceId="+groupInstanceID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("nested recap status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire TaskRecap
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.TaskID != "register" || len(wire.Links) != 2 || wire.UserID != "ada" {
		t.Fatalf("wire = %#v", wire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("flat recap = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=register&jobInstanceId="+truckID+
			"&groupId=lots&groupInstanceId=ghost", nil))
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
