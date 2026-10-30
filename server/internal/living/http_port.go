package living

import "context"

type HttpAction string

const (
	HttpActionLink     HttpAction = "link"
	HttpActionDownload HttpAction = "download"
)

// HttpRequest is one outbound side-effect call from onSubmit.http.
type HttpRequest struct {
	Method string
	URL    string
	Body   any
}

// HttpPortResponse URL is a plain-text body; headers may carry Content-Disposition.
type HttpPortResponse struct {
	URL     string
	Headers map[string]string
}

// HttpLinkItem is one successful effect for the SPA link strip.
type HttpLinkItem struct {
	Label    string     `json:"label"`
	URL      string     `json:"url"`
	Action   HttpAction `json:"action"`
	Filename string     `json:"filename,omitempty"`
}

// HttpPort runs config-declared HTTP after a successful Domain mutate.
// Failures must not roll back Domain; the Committer collects errors on CommitResult.
type HttpPort interface {
	Request(ctx context.Context, req HttpRequest) (HttpPortResponse, error)
}
