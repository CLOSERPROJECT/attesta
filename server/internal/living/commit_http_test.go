package living

import (
	"errors"
	"testing"
	"time"
)

func TestCommitRegisterRunsOnSubmitHTTP(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port

	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"administration"},
	}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("domain rolled back? %#v", result.Domain.Load)
	}
	if len(port.calls) != 2 {
		t.Fatalf("http calls = %#v", port.calls)
	}
	if port.calls[0].Method != "POST" || port.calls[0].URL != "/fake/labels" {
		t.Fatalf("call0 = %#v", port.calls[0])
	}
	if port.calls[1].URL != "/fake/register-csv" {
		t.Fatalf("call1 = %#v", port.calls[1])
	}
	if len(result.Links) != 2 {
		t.Fatalf("links = %#v", result.Links)
	}
	if result.Links[0].Label != "Stampa etichette" || result.Links[0].Action != HttpActionLink || result.Links[0].URL != "blob:labels" {
		t.Fatalf("link0 = %#v", result.Links[0])
	}
	if result.Links[1].Action != HttpActionDownload || result.Links[1].Filename != "reg.csv" {
		t.Fatalf("link1 = %#v", result.Links[1])
	}
	if len(result.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("persisted domain = %#v", stored)
	}
}

func TestCommitHTTPFailureDoesNotRollbackDomain(t *testing.T) {
	port := &recordingHttpPort{
		errs: []error{errors.New("printer down"), errors.New("csv down")},
	}
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port

	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"administration"},
	}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when HttpPort fails: %v", err)
	}
	if result.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("domain missing after http failure: %#v", result.Domain.Load)
	}
	if len(result.Domain.Pallet) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Pallet)
	}
	if len(result.Links) != 0 {
		t.Fatalf("links = %#v", result.Links)
	}
	if len(result.HttpErrors) != 2 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}
	if result.HttpErrors[0] != "printer down" || result.HttpErrors[1] != "csv down" {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("DomainStore rolled back: %#v", stored)
	}
}

func TestCommitWithoutHTTPSkipsPort(t *testing.T) {
	port := &recordingHttpPort{}
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.Now = func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) }

	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"administration"},
	}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// codiceFir has transform only — confirm a non-http Task does not call the port.
	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"administration"},
	}, CommitRequest{
		TaskPath:      "intake/codiceFir",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         map[string]any{"codiceFir": "FIR-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.calls) != 2 {
		t.Fatalf("register should call http twice; codiceFir zero more — calls=%d", len(port.calls))
	}
	if len(result.Links) != 0 || len(result.HttpErrors) != 0 {
		t.Fatalf("codiceFir http fields = links %#v errors %#v", result.Links, result.HttpErrors)
	}
}
