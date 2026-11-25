package org

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadTaskRecapFactsWithoutLinks(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	table, err := loader.LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "truckArrive",
	})
	if err != nil {
		t.Fatal(err)
	}
	if table.Mode != taskRecapModeSubmission {
		t.Fatalf("mode = %q", table.Mode)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("rows = %#v", table.Rows)
	}
	row := table.Rows[0]
	if row.UserID != "ada" || row.Timestamp == "" || row.SubmissionID != row.Timestamp {
		t.Fatalf("actor/time = %#v", row)
	}
	if row.UserName != "ada" {
		t.Fatalf("userName = %#v (want UserID fallback)", row.UserName)
	}
	if row.Cells["truckWeight"] != 1000.0 {
		t.Fatalf("cells = %#v", row.Cells)
	}
	if _, has := row.Cells["date"]; has {
		t.Fatalf("truckArrive facts must omit date: %#v", row.Cells)
	}
	if _, has := row.Cells["turno"]; has {
		t.Fatalf("truckArrive facts must omit turno: %#v", row.Cells)
	}
	col := columnByKey(table.Columns, "truckWeight")
	if col == nil || col.Title != "Peso lordo camion" {
		t.Fatalf("columns = %#v", table.Columns)
	}
	if len(row.Links) != 0 {
		t.Fatalf("truckArrive has no links: %#v", row.Links)
	}
}

func TestLoadTaskRecapManySubmitEntityModeWithPerRowLinks(t *testing.T) {
	clock := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels-1"},
			{URL: "blob:labels-2"},
		},
	}
	links := NewMemoryLinkStore()
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
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
	table, err := loader.LoadTaskRecap(t.Context(), FeragameScope(), addr)
	if err != nil {
		t.Fatal(err)
	}
	if table.Mode != taskRecapModeEntity {
		t.Fatalf("mode = %q", table.Mode)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows = %#v", table.Rows)
	}
	if table.Rows[0].Timestamp >= table.Rows[1].Timestamp {
		t.Fatalf("want oldest first: %q then %q", table.Rows[0].Timestamp, table.Rows[1].Timestamp)
	}
	if table.Rows[0].Cells["client"] != "CO" || table.Rows[1].Cells["client"] != "DA" {
		t.Fatalf("cells order = %#v %#v", table.Rows[0].Cells, table.Rows[1].Cells)
	}
	if len(table.Rows[0].Links) != 1 || table.Rows[0].Links[0].URL != "blob:labels-1" {
		t.Fatalf("first links = %#v", table.Rows[0].Links)
	}
	if len(table.Rows[1].Links) != 1 || table.Rows[1].Links[0].URL != "blob:labels-2" {
		t.Fatalf("second links = %#v", table.Rows[1].Links)
	}
	col := columnByKey(table.Columns, "client")
	if col == nil || col.Title != "Cliente" {
		t.Fatalf("columns = %#v", table.Columns)
	}
}

func TestLoadTaskRecapRejectsNotDoneAndUnknown(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)

	_, err := loader.LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "register",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("not-done register = %v, want ErrNotFound", err)
	}

	_, err = loader.LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{
		JobID: "intake", JobInstanceID: "no-such", TaskID: "truckArrive",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown instance = %v, want ErrNotFound", err)
	}

	_, err = loader.LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{
		JobID: "intake", JobInstanceID: truckID, TaskID: "",
	})
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid address err = %v", err)
	}
}

func TestHandleTaskRecapHTTP(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
		},
	}
	links := NewMemoryLinkStore()
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = links
	loader.LinkStore = links

	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	h := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return admin, true
		},
	}

	truckID := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckID, registerLotValue())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=register&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("register recap status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire TaskRecapTable
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Mode != taskRecapModeEntity || len(wire.Rows) != 1 ||
		len(wire.Rows[0].Links) != 1 || wire.Rows[0].UserID != "ada" {
		t.Fatalf("wire = %#v", wire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("flat recap = %d", rec.Code)
	}
	var onceWire TaskRecapTable
	if err := json.Unmarshal(rec.Body.Bytes(), &onceWire); err != nil {
		t.Fatal(err)
	}
	if onceWire.Mode != taskRecapModeSubmission || len(onceWire.Rows) != 1 {
		t.Fatalf("once wire = %#v", onceWire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=weightCheck&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not done = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-recap?stream=recycling&instance="+truckID+"&taskId=truckArrive", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing params = %d", rec.Code)
	}

	// Load-level: any org Gate identity may read; no Cerbos Commit on this path.
	logH := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}, true
		},
	}
	rec = httptest.NewRecorder()
	logH.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("logistic recap = %d body=%s", rec.Code, rec.Body.String())
	}
}

func columnByKey(cols []TaskRecapCol, key string) *TaskRecapCol {
	for i := range cols {
		if cols[i].Key == key {
			return &cols[i]
		}
	}
	return nil
}
