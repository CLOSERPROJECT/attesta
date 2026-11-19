package main

import (
	"net/http"
	"os"
	"strings"
)

// buildInfoRevision is a variable so tests can exercise the unavailable case.
var buildInfoRevision = revisionFromBuildInfo

// handleRevision answers the deployed commit for the UAT gate's preview check.
func handleRevision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	revision := deployedRevision()
	if revision == "" {
		http.Error(w, "deployed revision is unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(revision + "\n"))
}

func deployedRevision() string {
	for _, candidate := range []string{
		os.Getenv("SOURCE_COMMIT"),
		buildCommit,
		buildInfoRevision(),
	} {
		candidate = strings.TrimSpace(candidate)
		if isFullRevision(candidate) {
			return strings.ToLower(candidate)
		}
	}
	return ""
}

func isFullRevision(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}
