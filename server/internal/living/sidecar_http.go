package living

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// ArtifactProxyPathPrefix is the browser-facing path Commit Links rewrite to.
	ArtifactProxyPathPrefix = "/api/living/artifacts/"
	sidecarArtifactPrefix   = "/artifacts/"
	attestaOrgHeader        = "X-Attesta-Org"
	defaultSidecarTimeout   = 8 * time.Second
)

// SidecarHttpPort is the product HttpPort Adapter for the living-http Compose peer.
type SidecarHttpPort struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// UnconfiguredHttpPort returns a soft error per Request so Domain Commit still succeeds.
type UnconfiguredHttpPort struct{}

func (UnconfiguredHttpPort) Request(context.Context, HttpRequest) (HttpPortResponse, error) {
	return HttpPortResponse{}, fmt.Errorf("living-http not configured")
}

// HttpPortFromEnv builds the product port: real sidecar client, or soft-fail when URL unset.
func HttpPortFromEnv() HttpPort {
	base := strings.TrimSpace(os.Getenv("LIVING_HTTP_URL"))
	if base == "" {
		return UnconfiguredHttpPort{}
	}
	return NewSidecarHttpPort(base, strings.TrimSpace(os.Getenv("LIVING_HTTP_TOKEN")), nil)
}

func NewSidecarHttpPort(baseURL, token string, client *http.Client) *SidecarHttpPort {
	if client == nil {
		client = &http.Client{Timeout: defaultSidecarTimeout}
	}
	return &SidecarHttpPort{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
		HTTP:    client,
	}
}

func (p *SidecarHttpPort) Request(ctx context.Context, req HttpRequest) (HttpPortResponse, error) {
	if p == nil || p.BaseURL == "" {
		return HttpPortResponse{}, fmt.Errorf("living-http not configured")
	}
	path, err := requirePathOnlyURL(req.URL)
	if err != nil {
		return HttpPortResponse{}, err
	}
	target, err := joinBasePath(p.BaseURL, path)
	if err != nil {
		return HttpPortResponse{}, err
	}

	var bodyReader io.Reader
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if req.Body != nil {
		raw, err := json.Marshal(req.Body)
		if err != nil {
			return HttpPortResponse{}, fmt.Errorf("encode http body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
	if err != nil {
		return HttpPortResponse{}, err
	}
	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if p.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.Token)
	}
	if org := strings.TrimSpace(req.Org); org != "" {
		httpReq.Header.Set(attestaOrgHeader, org)
	}

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return HttpPortResponse{}, fmt.Errorf("living-http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return HttpPortResponse{}, fmt.Errorf("living-http read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return HttpPortResponse{}, fmt.Errorf("living-http status %d", resp.StatusCode)
	}
	plain := strings.TrimSpace(string(body))
	headers := map[string]string{}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		headers["Content-Disposition"] = cd
	}
	return HttpPortResponse{
		URL:     rewriteSidecarArtifactURL(plain),
		Headers: headers,
	}, nil
}

// Artifact holds a sidecar GET payload for the Attesta browser proxy.
type Artifact struct {
	OrgID       string
	ContentType string
	Filename    string
	Body        []byte
	Disposition string
}

// FetchArtifact loads one artifact from the sidecar with the shared secret.
func (p *SidecarHttpPort) FetchArtifact(ctx context.Context, id string) (Artifact, error) {
	if p == nil || p.BaseURL == "" {
		return Artifact{}, fmt.Errorf("living-http not configured")
	}
	id = strings.TrimSpace(id)
	if !validArtifactID(id) {
		return Artifact{}, fmt.Errorf("invalid artifact id")
	}
	target, err := joinBasePath(p.BaseURL, sidecarArtifactPrefix+id)
	if err != nil {
		return Artifact{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Artifact{}, err
	}
	if p.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.Token)
	}
	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return Artifact{}, fmt.Errorf("living-http fetch: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return Artifact{}, fmt.Errorf("living-http read artifact: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return Artifact{}, fmt.Errorf("artifact not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Artifact{}, fmt.Errorf("living-http status %d", resp.StatusCode)
	}
	art := Artifact{
		OrgID:       strings.TrimSpace(resp.Header.Get(attestaOrgHeader)),
		ContentType: resp.Header.Get("Content-Type"),
		Disposition: resp.Header.Get("Content-Disposition"),
		Body:        body,
	}
	if art.Disposition != "" {
		art.Filename = filenameFromContentDisposition(art.Disposition)
	}
	return art, nil
}

func rewriteSidecarArtifactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, sidecarArtifactPrefix) {
		id := strings.TrimPrefix(raw, sidecarArtifactPrefix)
		if validArtifactID(id) {
			return ArtifactProxyPathPrefix + id
		}
	}
	return raw
}

// requirePathOnlyURL rejects absolute http(s) and protocol-relative URLs (SSRF lock).
func requirePathOnlyURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("http.url must evaluate to a non-empty string")
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(raw, "//") {
		return "", fmt.Errorf("http.url must be a path, not an absolute URL")
	}
	if !strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("http.url must be an absolute path")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("http.url is not a valid path: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return "", fmt.Errorf("http.url must be a path, not an absolute URL")
	}
	return raw, nil
}

func joinBasePath(base, path string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("living-http base URL is empty")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("living-http base URL is invalid")
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("http.url is not a valid path: %w", err)
	}
	return u.ResolveReference(ref).String(), nil
}

func validArtifactID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
