package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	orgHeader          = "X-Attesta-Org"
	maxBodyBytes       = 1 << 20
	artifactPathPrefix = "/artifacts/"
)

type server struct {
	store *diskStore
	token string
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /fake/labels", s.handleLabels)
	mux.HandleFunc("POST /fake/register-csv", s.handleRegisterCSV)
	mux.HandleFunc("GET /artifacts/{id}", s.handleGetArtifact)
	return s.withAuth(s.withAccessLog(mux))
}

func (s *server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := bearerToken(r.Header.Get("Authorization"))
		if got == "" || got != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		org := strings.TrimSpace(r.Header.Get(orgHeader))
		log.Printf("method=%s path=%s status=%d duration=%s org=%q",
			r.Method, r.URL.Path, rw.status, time.Since(start).Round(time.Millisecond), org)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func (s *server) handleLabels(w http.ResponseWriter, r *http.Request) {
	body, err := decodeJSONObject(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pdf, err := labelsPDF(body)
	if err != nil {
		http.Error(w, "pdf failed", http.StatusInternalServerError)
		return
	}
	id, err := s.store.put(r.Header.Get(orgHeader), "application/pdf", "", pdf)
	if err != nil {
		http.Error(w, "store failed", http.StatusInternalServerError)
		return
	}
	writeArtifactURL(w, id)
}

func (s *server) handleRegisterCSV(w http.ResponseWriter, r *http.Request) {
	body, err := decodeJSONObject(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bytes, filename := registerCSV(body)
	id, err := s.store.put(r.Header.Get(orgHeader), "text/csv", filename, bytes)
	if err != nil {
		http.Error(w, "store failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	writeArtifactURL(w, id)
}

func (s *server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	meta, data, err := s.store.get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if meta.ContentType != "" {
		w.Header().Set("Content-Type", meta.ContentType)
	}
	if meta.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.Filename))
	}
	if org := strings.TrimSpace(meta.OrgID); org != "" {
		w.Header().Set(orgHeader, org)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// HttpPort reads the response body as a plain-text URL; keep path-only so 01B
// can rewrite without trusting a sidecar host embedded in Commit Links.
func writeArtifactURL(w http.ResponseWriter, id string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, artifactPathPrefix+id)
}

func decodeJSONObject(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxBodyBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(raw) > maxBodyBytes {
		return nil, fmt.Errorf("body too large")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, nil
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("body must be a JSON object")
	}
	if body == nil {
		return map[string]any{}, nil
	}
	return body, nil
}
