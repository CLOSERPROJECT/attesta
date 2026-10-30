package living

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SubmitHttpEffect is one authored onSubmit.http entry (exprs still raw).
type SubmitHttpEffect struct {
	Method  any
	URL     any
	Body    any
	Label   any
	Action  any
	HasBody bool
}

type runSubmitHTTPResult struct {
	Links  []HttpLinkItem
	Errors []string
}

func taskOnSubmitHTTP(task map[string]any) []SubmitHttpEffect {
	onSubmit, _ := task["onSubmit"].(map[string]any)
	if onSubmit == nil {
		return nil
	}
	return parseSubmitHTTP(onSubmit["http"])
}

func parseSubmitHTTP(raw any) []SubmitHttpEffect {
	if raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]SubmitHttpEffect, 0, len(list))
	for _, item := range list {
		rec, ok := item.(map[string]any)
		if !ok {
			return nil
		}
		if _, has := rec["method"]; !has {
			return nil
		}
		if _, has := rec["url"]; !has {
			return nil
		}
		if _, has := rec["label"]; !has {
			return nil
		}
		if _, has := rec["action"]; !has {
			return nil
		}
		effect := SubmitHttpEffect{
			Method: rec["method"],
			URL:    rec["url"],
			Label:  rec["label"],
			Action: rec["action"],
		}
		if body, has := rec["body"]; has {
			effect.Body = body
			effect.HasBody = true
		}
		out = append(out, effect)
	}
	return out
}

// runSubmitHTTPEffects resolves onSubmit.http, calls HttpPort in order, collects links.
// Failures accumulate; remaining effects still run. Never rolls back Domain.
func runSubmitHTTPEffects(
	ctx context.Context,
	port HttpPort,
	effects []SubmitHttpEffect,
	facts map[string]any,
	bindings ExprBindings,
	domain Domain,
	jobInstanceID string,
) runSubmitHTTPResult {
	if len(effects) == 0 {
		return runSubmitHTTPResult{}
	}
	merged := mergeBindings(domainBindings(domain, jobInstanceID), bindings)
	var errors []string
	var links []HttpLinkItem

	for _, effect := range effects {
		item, errMsg := runOneSubmitHTTP(ctx, port, effect, facts, merged)
		if errMsg != "" {
			errors = append(errors, errMsg)
			continue
		}
		if item != nil {
			links = append(links, *item)
		}
	}
	return runSubmitHTTPResult{Links: links, Errors: errors}
}

func runOneSubmitHTTP(
	ctx context.Context,
	port HttpPort,
	effect SubmitHttpEffect,
	facts map[string]any,
	bindings ExprBindings,
) (*HttpLinkItem, string) {
	methodGot, err := evalScalar(effect.Method, facts, bindings)
	if err != nil {
		return nil, err.Error()
	}
	method, err := requireHTTPString(methodGot.Value, "http.method")
	if err != nil {
		return nil, err.Error()
	}

	urlGot, err := evalScalar(effect.URL, facts, bindings)
	if err != nil {
		return nil, err.Error()
	}
	reqURL, err := requireHTTPString(urlGot.Value, "http.url")
	if err != nil {
		return nil, err.Error()
	}

	labelGot, err := evalScalar(effect.Label, facts, bindings)
	if err != nil {
		return nil, err.Error()
	}
	label, err := requireHTTPString(labelGot.Value, "http.label")
	if err != nil {
		return nil, err.Error()
	}

	// YAML action: link|download is a plain enum — do not JSONata-path it.
	var actionRaw any
	if s, ok := effect.Action.(string); ok && (s == string(HttpActionLink) || s == string(HttpActionDownload)) {
		actionRaw = s
	} else {
		actionGot, err := evalScalar(effect.Action, facts, bindings)
		if err != nil {
			return nil, err.Error()
		}
		actionRaw = actionGot.Value
	}
	action, err := requireHTTPAction(actionRaw)
	if err != nil {
		return nil, err.Error()
	}

	req := HttpRequest{Method: method, URL: reqURL}
	if effect.HasBody {
		bodyGot, err := evalScalar(effect.Body, facts, bindings)
		if err != nil {
			return nil, err.Error()
		}
		req.Body = bodyGot.Value
	}

	// Committer skips the whole effects block when HttpPort is nil; this guard is defensive.
	if port == nil {
		return nil, "http port not configured"
	}
	resp, err := port.Request(ctx, req)
	if err != nil {
		return nil, err.Error()
	}
	respURL, err := requirePlainTextURL(resp.URL)
	if err != nil {
		return nil, err.Error()
	}

	item := &HttpLinkItem{Label: label, URL: respURL, Action: action}
	if action == HttpActionDownload {
		if filename := filenameFromContentDisposition(headerValue(resp.Headers, "Content-Disposition")); filename != "" {
			item.Filename = filename
		}
	}
	return item, ""
}

func requireHTTPString(value any, label string) (string, error) {
	s, ok := value.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("%s must evaluate to a non-empty string", label)
	}
	return s, nil
}

func requireHTTPAction(value any) (HttpAction, error) {
	s, ok := value.(string)
	if !ok || (s != string(HttpActionLink) && s != string(HttpActionDownload)) {
		return "", fmt.Errorf("http.action must be %q or %q", HttpActionLink, HttpActionDownload)
	}
	return HttpAction(s), nil
}

func requirePlainTextURL(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("http response body must be a non-empty plain-text URL")
	}
	if strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("http response body must be a single-line plain-text URL")
	}
	return value, nil
}

func headerValue(headers map[string]string, name string) string {
	if headers == nil {
		return ""
	}
	if v, ok := headers[name]; ok {
		return v
	}
	lower := strings.ToLower(name)
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

var (
	contentDispositionStar  = regexp.MustCompile(`(?i)filename\*\s*=\s*(?:UTF-8''|utf-8'')([^;]+)`)
	contentDispositionPlain = regexp.MustCompile(`(?i)filename\s*=\s*("(?:\\.|[^"])*"|[^;]+)`)
)

func filenameFromContentDisposition(header string) string {
	if header == "" {
		return ""
	}
	if star := contentDispositionStar.FindStringSubmatch(header); len(star) > 1 {
		raw := strings.TrimSpace(star[1])
		raw = strings.Trim(raw, `"'`)
		if decoded, err := url.QueryUnescape(raw); err == nil {
			return decoded
		}
		return raw
	}
	plain := contentDispositionPlain.FindStringSubmatch(header)
	if len(plain) < 2 {
		return ""
	}
	raw := strings.TrimSpace(plain[1])
	if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) && len(raw) >= 2 {
		return strings.ReplaceAll(raw[1:len(raw)-1], `\"`, `"`)
	}
	return raw
}
