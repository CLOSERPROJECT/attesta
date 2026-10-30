package living

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
)

// FakeHttpPort is the plant stand-in for fixture /fake/* onSubmit.http endpoints.
// Labels return a minimal PDF data URL; register-csv returns a CSV data URL with disposition.
type FakeHttpPort struct {
	seq atomic.Uint64
}

func NewFakeHttpPort() *FakeHttpPort {
	return &FakeHttpPort{}
}

func (f *FakeHttpPort) Request(_ context.Context, req HttpRequest) (HttpPortResponse, error) {
	path := fakeHTTPPath(req.URL)
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	switch {
	case method == "POST" && path == "/fake/labels":
		_ = f.seq.Add(1)
		return HttpPortResponse{
			URL: "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(minimalLabelsPDF(req.Body)),
		}, nil
	case method == "POST" && path == "/fake/register-csv":
		body, ok := req.Body.(map[string]any)
		if !ok {
			return HttpPortResponse{}, fmt.Errorf("register-csv body must be an object")
		}
		bytes, filename := registerCSV(body)
		_ = f.seq.Add(1)
		return HttpPortResponse{
			URL: "data:text/csv;base64," + base64.StdEncoding.EncodeToString(bytes),
			Headers: map[string]string{
				"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, filename),
			},
		}, nil
	default:
		return HttpPortResponse{}, fmt.Errorf("unknown fake http endpoint: %s %s", req.Method, req.URL)
	}
}

func fakeHTTPPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		return u.Path
	}
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		return raw[:i]
	}
	return raw
}

func registerCSV(body map[string]any) ([]byte, string) {
	headers := []string{
		"client", "date", "wasteCode", "truckWeight", "truckTare",
		"palletCount", "notes", "loadId", "palletIds",
	}
	palletIds := ""
	if raw, ok := body["palletIds"]; ok {
		switch v := raw.(type) {
		case []any:
			parts := make([]string, 0, len(v))
			for _, id := range v {
				parts = append(parts, fmt.Sprint(id))
			}
			palletIds = strings.Join(parts, " ")
		case []string:
			palletIds = strings.Join(v, " ")
		default:
			palletIds = fmt.Sprint(v)
		}
	}
	row := make([]string, len(headers))
	for i, h := range headers {
		if h == "palletIds" {
			row[i] = csvEscape(palletIds)
			continue
		}
		row[i] = csvEscape(body[h])
	}
	text := strings.Join(headers, ",") + "\n" + strings.Join(row, ",") + "\n"
	loadID := "register"
	if s, ok := body["loadId"].(string); ok && s != "" {
		loadID = s
	}
	return []byte(text), loadID + ".csv"
}

func csvEscape(value any) string {
	if value == nil {
		return ""
	}
	s := fmt.Sprint(value)
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// Tiny valid-enough PDF so the link strip has a real data URL without a template file.
func minimalLabelsPDF(body any) []byte {
	label := "labels"
	if rec, ok := body.(map[string]any); ok {
		if id, ok := rec["loadId"].(string); ok && id != "" {
			label = id
		}
	}
	// Keep content ASCII and short; browsers still open data:application/pdf.
	return []byte("%PDF-1.1\n1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n" +
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n" +
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>endobj\n" +
		"trailer<< /Root 1 0 R /Info << /Title (" + label + ") >> >>\n%%EOF\n")
}
