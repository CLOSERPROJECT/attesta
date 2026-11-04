package living

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestFakeHttpPortLabelsAndRegisterCSV(t *testing.T) {
	port := NewFakeHttpPort()
	labels, err := port.Request(t.Context(), HttpRequest{
		Method: "POST",
		URL:    "/fake/labels",
		Body: map[string]any{
			"loadId": "NIR-1",
			"labels": []any{map[string]any{"id": "NIR-1.1"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(labels.URL, "data:application/pdf;base64,") {
		t.Fatalf("labels url = %q", labels.URL)
	}

	csvResp, err := port.Request(t.Context(), HttpRequest{
		Method: "POST",
		URL:    "/fake/register-csv",
		Body: map[string]any{
			"client":    "NO",
			"loadId":    "NIR-1",
			"palletIds": []any{"NIR-1.1", "NIR-1.2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(csvResp.URL, "data:text/csv;base64,") {
		t.Fatalf("csv url = %q", csvResp.URL)
	}
	if got := csvResp.Headers["Content-Disposition"]; got != `attachment; filename="NIR-1.csv"` {
		t.Fatalf("disposition = %q", got)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(csvResp.URL, "data:text/csv;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "NIR-1.1 NIR-1.2") || !strings.HasPrefix(text, "client,date,") {
		t.Fatalf("csv body = %q", text)
	}
}

func TestFakeHttpPortUnknownEndpoint(t *testing.T) {
	_, err := NewFakeHttpPort().Request(t.Context(), HttpRequest{Method: "GET", URL: "/fake/other"})
	if err == nil || !strings.Contains(err.Error(), "unknown fake http endpoint") {
		t.Fatalf("err = %v", err)
	}
}
