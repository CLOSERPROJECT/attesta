package living

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// OperatorIdentity is the verified session actor injected by the host.
type OperatorIdentity struct {
	UserID string
	Org    string
	Roles  []string
}

// Gate authenticates and verifies the caller. It writes the HTTP error response
// and returns ok=false when the request must not proceed.
type Gate func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool)

// Handler serves living Load (and later Commit) over placeholder HTTP seams.
type Handler struct {
	Loader *Loader
	Gate   Gate
}

// ServeHTTP routes GET /api/living/domain (and method-not-allowed otherwise).
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/living/domain" || strings.TrimSuffix(r.URL.Path, "/") == "/api/living/domain":
		h.handleLoad(w, r)
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
