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
	Org    string
	Roles  []string
}

// Gate authenticates the caller; on failure it writes the HTTP error and returns ok=false.
type Gate func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool)

// Handler serves Load and Commit over placeholder HTTP seams.
type Handler struct {
	Loader    *Loader
	Committer *Committer
	Gate      Gate
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch path {
	case "/api/living/domain":
		h.handleLoad(w, r)
	case "/api/living/commit":
		h.handleCommit(w, r)
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

	result, err := h.Loader.Load(r.Context(), org)
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
