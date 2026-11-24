package org

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultEventsHeartbeat = 25 * time.Second

// handleEvents streams Domain revisions for one Org stream as Server-Sent Events.
// Events carry no Domain data: clients re-Load, and the server projects (ADR-0005).
func (h *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Gate == nil || h.DomainLoader == nil || h.DomainLoader.Domains == nil || h.Changes == nil {
		http.Error(w, "org events not configured", http.StatusInternalServerError)
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
	streamKey := strings.TrimSpace(r.URL.Query().Get("stream"))
	if streamKey == "" {
		http.Error(w, "stream required", http.StatusBadRequest)
		return
	}
	scope, _, ok := h.resolveStreamScope(w, r.Context(), org, streamKey, h.DomainLoader.Configs)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	revisions, cancel := h.Changes.Subscribe(scope)
	defer cancel()

	// Subscribing before the revision read means a write landing between the two
	// calls arrives as an event instead of being lost.
	current, err := h.DomainLoader.Domains.Revision(r.Context(), scope)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			http.Error(w, "failed to read domain revision", http.StatusInternalServerError)
			return
		}
		current = 0
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	_, _ = io.WriteString(w, "retry: 5000\n\n")
	writeDomainRevisionEvent(w, current)
	flusher.Flush()

	heartbeat := h.EventsHeartbeat
	if heartbeat <= 0 {
		heartbeat = defaultEventsHeartbeat
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case revision, open := <-revisions:
			if !open {
				return
			}
			writeDomainRevisionEvent(w, revision)
			flusher.Flush()
		case <-ticker.C:
			_, _ = io.WriteString(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeDomainRevisionEvent(w io.Writer, revision int64) {
	_, _ = fmt.Fprintf(w, "event: domain-revision\ndata: {\"revision\":%d}\n\n", revision)
}
