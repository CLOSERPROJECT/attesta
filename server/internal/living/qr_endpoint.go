package living

import (
	"os"
	"strings"
)

// LivingAppBaseURL is the Living SPA origin: PUBLIC_BASE_URL (trim trailing /) + "/app".
// Empty when PUBLIC_BASE_URL is unset. Bound as $qrEndpoint for onSubmit.http.
func LivingAppBaseURL() string {
	base := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/app"
}

// SubmitBindingsFromEnv builds ops JSONata vars for onSubmit.http (e.g. $qrEndpoint).
// Nil when nothing to inject.
func SubmitBindingsFromEnv() ExprBindings {
	if ep := LivingAppBaseURL(); ep != "" {
		return ExprBindings{"qrEndpoint": ep}
	}
	return nil
}
