package org

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// OperatorIdentity is the verified session actor injected by the host.
type OperatorIdentity struct {
	UserID string
	// DisplayName is human-facing chrome ($operator.displayName); email when known.
	DisplayName string
	Org         string
	Roles       []string
}

// Gate authenticates the caller; on failure it writes the HTTP error and returns ok=false.
type Gate func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool)

// API serves Load, addressed Task form / recap, Commit, QR resolve, and artifact proxy over HTTP seams.
type API struct {
	DomainLoader    *DomainLoader
	TaskCommitter *TaskCommitter
	Gate      Gate
	// Artifacts fetches sidecar bytes with the shared secret; nil → artifact route unavailable.
	Artifacts *SidecarHttpPort
}

func (h *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case path == "/api/org/domain":
		h.handleLoad(w, r)
	case path == "/api/org/task-form":
		h.handleTaskForm(w, r)
	case path == "/api/org/task-recap":
		h.handleTaskRecap(w, r)
	case path == "/api/org/commit":
		h.handleCommit(w, r)
	case path == "/api/org/qr-resolve":
		h.handleQrResolve(w, r)
	case strings.HasPrefix(path, ArtifactProxyPathPrefix):
		h.handleArtifact(w, r, strings.TrimPrefix(path, ArtifactProxyPathPrefix))
	default:
		http.NotFound(w, r)
	}
}

func (h *API) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.DomainLoader == nil {
		http.Error(w, "org load not configured", http.StatusInternalServerError)
		return
	}
	op, ok := h.Gate(w, r)
	if !ok {
		return
	}
	org := strings.TrimSpace(op.Org)
	if org == "" {
		http.Error(w, "organization required", http.StatusForbidden)
		return
	}

	result, err := h.DomainLoader.Load(r.Context(), org, op)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "active config not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load domain", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		return
	}
}

func (h *API) handleTaskForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.DomainLoader == nil {
		http.Error(w, "org task-form not configured", http.StatusInternalServerError)
		return
	}
	op, ok := h.Gate(w, r)
	if !ok {
		return
	}
	org := strings.TrimSpace(op.Org)
	if org == "" {
		http.Error(w, "organization required", http.StatusForbidden)
		return
	}
	q := r.URL.Query()
	jobID := strings.TrimSpace(q.Get("jobId"))
	taskID := strings.TrimSpace(q.Get("taskId"))
	jobInstanceID := q.Get("jobInstanceId")
	if jobID == "" || taskID == "" {
		http.Error(w, "jobId and taskId required", http.StatusBadRequest)
		return
	}

	bundle, err := h.DomainLoader.LoadTaskBundle(r.Context(), org, op, jobID, taskID, jobInstanceID)
	if err != nil {
		if errors.Is(err, ErrTaskBundleForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "active config not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load task form", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(bundle)
}

func (h *API) handleTaskRecap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.DomainLoader == nil {
		http.Error(w, "org task-recap not configured", http.StatusInternalServerError)
		return
	}
	op, ok := h.Gate(w, r)
	if !ok {
		return
	}
	org := strings.TrimSpace(op.Org)
	if org == "" {
		http.Error(w, "organization required", http.StatusForbidden)
		return
	}
	q := r.URL.Query()
	addr := SubmissionAddress{
		JobID:         strings.TrimSpace(q.Get("jobId")),
		JobInstanceID: strings.TrimSpace(q.Get("jobInstanceId")),
		TaskID:        strings.TrimSpace(q.Get("taskId")),
	}
	if addr.JobID == "" || addr.TaskID == "" || addr.JobInstanceID == "" {
		http.Error(w, "jobId, jobInstanceId, and taskId required", http.StatusBadRequest)
		return
	}

	list, err := h.DomainLoader.LoadTaskRecap(r.Context(), org, addr)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "task recap not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "submission address required") {
			http.Error(w, "invalid submission address", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to load task recap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(list)
}

func (h *API) handleCommit(w http.ResponseWriter, r *http.Request) {
	op, org, body, ok := h.preparePOSTAuth(w, r, "org commit not configured", h.TaskCommitter != nil)
	if !ok {
		return
	}

	var req CommitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := h.TaskCommitter.Commit(r.Context(), org, op, req)
	if err != nil {
		writeCommitError(w, err)
		return
	}
	writeCommitResult(w, result)
}

type qrResolveRequest struct {
	Raw string `json:"raw"`
}

func (h *API) handleQrResolve(w http.ResponseWriter, r *http.Request) {
	op, org, body, ok := h.preparePOSTAuth(w, r, "org qr-resolve not configured", h.DomainLoader != nil)
	if !ok {
		return
	}
	var req qrResolveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	loaded, err := h.DomainLoader.Load(r.Context(), org, op)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "active config not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to resolve qr", http.StatusInternalServerError)
		return
	}
	result := ResolveQR(loaded.Config, loaded.Domain, op, req.Raw)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if result.Error != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(result)
}

// preparePOSTAuth gates a POST, requires org, and reads the body (1 MiB).
// depOK is the route-specific dependency (TaskCommitter or DomainLoader); Gate is always required.
func (h *API) preparePOSTAuth(
	w http.ResponseWriter,
	r *http.Request,
	notConfigured string,
	depOK bool,
) (op OperatorIdentity, org string, body []byte, ok bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || !depOK {
		http.Error(w, notConfigured, http.StatusInternalServerError)
		return
	}
	op, gated := h.Gate(w, r)
	if !gated {
		return
	}
	org = strings.TrimSpace(op.Org)
	if org == "" {
		http.Error(w, "organization required", http.StatusForbidden)
		return
	}
	var err error
	body, err = io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	ok = true
	return
}

func writeCommitResult(w http.ResponseWriter, result CommitResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(result)
}

func (h *API) handleArtifact(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil {
		http.Error(w, "org artifacts not configured", http.StatusInternalServerError)
		return
	}
	op, ok := h.Gate(w, r)
	if !ok {
		return
	}
	org := strings.TrimSpace(op.Org)
	if org == "" {
		http.Error(w, "organization required", http.StatusForbidden)
		return
	}
	id = strings.TrimSpace(id)
	if !validArtifactID(id) {
		http.NotFound(w, r)
		return
	}
	if h.Artifacts == nil {
		http.Error(w, "org-http not configured", http.StatusServiceUnavailable)
		return
	}

	art, err := h.Artifacts.FetchArtifact(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid artifact") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to fetch artifact", http.StatusBadGateway)
		return
	}
	// Fail closed: artifact must be bound to this session's org at create time.
	if art.OrgID == "" || art.OrgID != org {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if art.ContentType != "" {
		w.Header().Set("Content-Type", art.ContentType)
	}
	if art.Disposition != "" {
		w.Header().Set("Content-Disposition", art.Disposition)
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(art.Body)
}

func writeCommitError(w http.ResponseWriter, err error) {
	var ce *CommitError
	if errors.As(err, &ce) {
		switch ce.Kind {
		case "unauthorized":
			http.Error(w, "forbidden", http.StatusForbidden)
		case "unknown-target", "prep":
			http.Error(w, ce.Message, http.StatusBadRequest)
		case "forbidden":
			http.Error(w, ce.Message, http.StatusForbidden)
		case "mutate", "passport-exists":
			http.Error(w, ce.Message, http.StatusConflict)
		default:
			http.Error(w, "commit failed", http.StatusInternalServerError)
		}
		return
	}
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "active config not found", http.StatusNotFound)
		return
	}
	http.Error(w, "failed to commit", http.StatusInternalServerError)
}
