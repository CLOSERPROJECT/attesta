package org

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newEventsTestAPI(t *testing.T) (*API, *DomainChangeHub, *TaskCommitter) {
	t.Helper()
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	hub := NewDomainChangeHub()
	c.Changes = hub
	api := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Changes:       hub,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			if r.Header.Get("X-Test-Denied") != "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return OperatorIdentity{}, false
			}
			op := adminOp()
			switch r.Header.Get("X-Test-Operator") {
			case "second":
				op.UserID = "bob"
				op.DisplayName = "Bob"
			case "empty-org":
				op.Org = ""
			}
			return op, true
		},
	}
	return api, hub, c
}

func openDomainEvents(t *testing.T, api *API, query string) (*bufio.Reader, *http.Response, func()) {
	t.Helper()
	srv := httptest.NewServer(api)
	ctx, cancelCtx := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/org/events"+query, nil)
	if err != nil {
		cancelCtx()
		srv.Close()
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancelCtx()
		srv.Close()
		t.Fatal(err)
	}
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			cancelCtx()
			_ = resp.Body.Close()
			srv.Close()
		})
	}
	t.Cleanup(cancel)
	return bufio.NewReader(resp.Body), resp, cancel
}

// readEventBlock returns the next block ended by a blank line, or fails the test
// after a timeout so a stalled stream cannot hang the package.
func readEventBlock(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	type result struct {
		block string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		var block strings.Builder
		for {
			line, err := reader.ReadString('\n')
			block.WriteString(line)
			if err != nil {
				done <- result{block.String(), err}
				return
			}
			if line == "\n" {
				done <- result{block.String(), nil}
				return
			}
		}
	}()
	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("read event block: %v (got %q)", res.err, res.block)
		}
		return res.block
	case <-time.After(2 * time.Second):
		t.Fatal("timed out reading event block")
		return ""
	}
}

func waitForSubscribers(t *testing.T, hub *DomainChangeHub, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		got := hub.subscriberCount(FeragameScope())
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("subscriberCount = %d, want %d", got, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHandleEventsStreamsDomainRevisions(t *testing.T) {
	api, hub, _ := newEventsTestAPI(t)
	body, resp, cancel := openDomainEvents(t, api, "?stream=recycling")
	defer cancel()

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := resp.Header.Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("X-Accel-Buffering = %q", got)
	}
	if got := readEventBlock(t, body); got != "retry: 5000\n\n" {
		t.Fatalf("retry block = %q", got)
	}
	// Fresh store: no Domain yet, so the current revision is 0.
	if got := readEventBlock(t, body); got != "event: domain-revision\ndata: {\"revision\":0}\n\n" {
		t.Fatalf("first event = %q", got)
	}

	commitReq := httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling",
		strings.NewReader(`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`))
	commitReq.Header.Set("X-Test-Operator", "second")
	commitRec := httptest.NewRecorder()
	api.ServeHTTP(commitRec, commitReq)
	if commitRec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", commitRec.Code, commitRec.Body.String())
	}

	if got := readEventBlock(t, body); got != "event: domain-revision\ndata: {\"revision\":1}\n\n" {
		t.Fatalf("commit event = %q", got)
	}

	cancel()
	waitForSubscribers(t, hub, 0)
}

func TestHandleEventsHeartbeatKeepsStreamAlive(t *testing.T) {
	api, _, _ := newEventsTestAPI(t)
	api.EventsHeartbeat = 20 * time.Millisecond
	body, _, cancel := openDomainEvents(t, api, "?stream=recycling")
	defer cancel()

	_ = readEventBlock(t, body) // retry
	_ = readEventBlock(t, body) // current revision
	if got := readEventBlock(t, body); got != ": keepalive\n\n" {
		t.Fatalf("keepalive block = %q", got)
	}
}

func TestHandleEventsRejectsNonGET(t *testing.T) {
	api, _, _ := newEventsTestAPI(t)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/events?stream=recycling", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleEventsRequiresConfiguration(t *testing.T) {
	api, _, _ := newEventsTestAPI(t)
	api.Changes = nil
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/events?stream=recycling", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "org events not configured") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleEventsGateFailurePassesThrough(t *testing.T) {
	api, _, _ := newEventsTestAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/org/events?stream=recycling", nil)
	req.Header.Set("X-Test-Denied", "1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleEventsRequiresOrgStreamAndKnownStream(t *testing.T) {
	tests := []struct {
		name   string
		target string
		header string
		status int
	}{
		{name: "empty org", target: "/api/org/events?stream=recycling", header: "empty-org", status: http.StatusForbidden},
		{name: "missing stream", target: "/api/org/events", status: http.StatusBadRequest},
		{name: "unknown stream", target: "/api/org/events?stream=missing", status: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api, _, _ := newEventsTestAPI(t)
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			if tc.header != "" {
				req.Header.Set("X-Test-Operator", tc.header)
			}
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d body=%s, want %d", rec.Code, rec.Body.String(), tc.status)
			}
		})
	}
}

type revisionErrDomainStore struct{ DomainStore }

func (revisionErrDomainStore) Revision(context.Context, Scope) (int64, error) {
	return 0, errors.New("boom")
}

func TestHandleEventsRevisionReadFailure(t *testing.T) {
	api, _, _ := newEventsTestAPI(t)
	api.DomainLoader = &DomainLoader{
		Domains: revisionErrDomainStore{api.DomainLoader.Domains},
		Configs: api.DomainLoader.Configs,
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/events?stream=recycling", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "failed to read domain revision") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}
