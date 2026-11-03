package living

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

// Handler serves Load, addressed Task form bundle, Commit, and artifact proxy over HTTP seams.
type Handler struct {
	Loader    *Loader
	Committer *Committer
	Gate      Gate
	// Artifacts fetches sidecar bytes with the shared secret; nil → artifact route unavailable.
	Artifacts *SidecarHttpPort
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case path == "/api/living/domain":
		h.handleLoad(w, r)
	case path == "/api/living/task-form":
		h.handleTaskForm(w, r)
	case path == "/api/living/commit":
		h.handleCommit(w, r)
	case strings.HasPrefix(path, ArtifactProxyPathPrefix):
		h.handleArtifact(w, r, strings.TrimPrefix(path, ArtifactProxyPathPrefix))
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.Loader == nil {
		http.Error(w, "living load not configured", http.StatusInternalServerError)
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

	result, err := h.Loader.Load(r.Context(), org, op)
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

func (h *Handler) handleTaskForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.Loader == nil {
		http.Error(w, "living task-form not configured", http.StatusInternalServerError)
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

	bundle, err := h.Loader.LoadTaskBundle(r.Context(), org, op, jobID, taskID, jobInstanceID)
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

func (h *Handler) handleCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.Committer == nil {
		http.Error(w, "living commit not configured", http.StatusInternalServerError)
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	var req CommitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := h.Committer.Commit(r.Context(), org, op, req)
	if err != nil {
		writeCommitError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(result)
}

func (h *Handler) handleArtifact(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil {
		http.Error(w, "living artifacts not configured", http.StatusInternalServerError)
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
		http.Error(w, "living-http not configured", http.StatusServiceUnavailable)
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
		case "mutate":
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
