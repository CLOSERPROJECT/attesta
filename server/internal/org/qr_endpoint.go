package org

import (
	"os"
	"strings"
)

// OrgAppBaseURL is the Living SPA stream entry: PUBLIC_BASE_URL (trim trailing /)
// + "/app/{stream}". Empty when PUBLIC_BASE_URL or stream is unset. Bound as
// $qrEndpoint for onSubmit.http so cold scans land in the right stream.
func OrgAppBaseURL(stream string) string {
	base := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	stream = strings.TrimSpace(stream)
	if base == "" || stream == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/app/" + stream
}

// SubmitBindingsForStream builds ops JSONata vars for onSubmit.http (e.g. $qrEndpoint).
// Nil when nothing to inject.
func SubmitBindingsForStream(stream string) ExprBindings {
	if ep := OrgAppBaseURL(stream); ep != "" {
		return ExprBindings{"qrEndpoint": ep}
	}
	return nil
}
