package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/yaml.v3"
)

const formataBuilderChromeOverrides = `
:root {
	--attesta-ink: #12210c;
	--attesta-muted: #4f5d47;
	--attesta-bg: #f3f4ef;
	--attesta-panel: #ffffff;
	--attesta-accent: #1e6f5c;
	--attesta-accent-2: #d2a106;
	--attesta-danger: #c0392b;
	--attesta-border: #d9e0d0;
	--attesta-shadow: rgba(16, 26, 20, 0.08);
	--attesta-page-bg: linear-gradient(135deg, #f3f4ef 0%, #e9efe2 100%);
	--attesta-chrome-bg: #fffaf2;
	--attesta-surface-2: #f8f9f4;
	--attesta-pill-bg: #eff4e9;

	--font-sans: "Space Grotesk", system-ui, sans-serif;
	--background: var(--attesta-bg);
	--foreground: var(--attesta-ink);
	--card: var(--attesta-panel);
	--card-foreground: var(--attesta-ink);
	--popover: var(--attesta-bg);
	--popover-foreground: var(--attesta-ink);
	--primary: var(--attesta-accent);
	--primary-foreground: var(--attesta-panel);
	--secondary: var(--attesta-surface-2);
	--secondary-foreground: var(--attesta-ink);
	--muted: var(--attesta-surface-2);
	--muted-foreground: var(--attesta-muted);
	--accent: var(--attesta-pill-bg);
	--accent-foreground: var(--attesta-ink);
	--destructive: var(--attesta-danger);
	--border: var(--attesta-border);
	--global-border: var(--attesta-border);
	--input: var(--attesta-panel);
	--ring: var(--attesta-accent);
	--sidebar: var(--attesta-bg);
	--sidebar-foreground: var(--attesta-ink);
	--sidebar-primary: var(--attesta-accent);
	--sidebar-primary-foreground: var(--attesta-panel);
	--sidebar-accent: var(--attesta-surface-2);
	--sidebar-accent-foreground: var(--attesta-ink);
	--sidebar-border: var(--attesta-border);
	--sidebar-ring: var(--attesta-accent);
	--warning: var(--attesta-accent-2);
}

.dark {
	--attesta-ink: #ecf1eb;
	--attesta-muted: #a6b2a2;
	--attesta-bg: #101411;
	--attesta-panel: #151c17;
	--attesta-accent: #57c2a6;
	--attesta-accent-2: #f2c64f;
	--attesta-danger: #ff6b5f;
	--attesta-border: rgba(236, 241, 235, 0.14);
	--attesta-shadow: rgba(0, 0, 0, 0.4);
	--attesta-page-bg: linear-gradient(135deg, #0b0f0d 0%, #121915 100%);
	--attesta-chrome-bg: #0f1411;
	--attesta-surface-2: rgba(255, 255, 255, 0.04);
	--attesta-pill-bg: rgba(255, 255, 255, 0.06);

	--background: var(--attesta-bg);
	--foreground: var(--attesta-ink);
	--card: var(--attesta-panel);
	--card-foreground: var(--attesta-ink);
	--popover: var(--attesta-bg);
	--popover-foreground: var(--attesta-ink);
	--primary: var(--attesta-accent);
	--primary-foreground: var(--attesta-panel);
	--secondary: var(--attesta-surface-2);
	--secondary-foreground: var(--attesta-ink);
	--muted: var(--attesta-surface-2);
	--muted-foreground: var(--attesta-muted);
	--accent: var(--attesta-pill-bg);
	--accent-foreground: var(--attesta-ink);
	--destructive: var(--attesta-danger);
	--border: var(--attesta-border);
	--global-border: var(--attesta-border);
	--input: var(--attesta-panel);
	--ring: var(--attesta-accent);
	--sidebar: var(--attesta-chrome-bg);
	--sidebar-foreground: var(--attesta-ink);
	--sidebar-primary: var(--attesta-accent);
	--sidebar-primary-foreground: var(--attesta-panel);
	--sidebar-accent: var(--attesta-surface-2);
	--sidebar-accent-foreground: var(--attesta-ink);
	--sidebar-border: var(--attesta-border);
	--sidebar-ring: var(--attesta-accent);
	--warning: var(--attesta-accent-2);
}

html,
body,
#app {
	font-family: "Space Grotesk", system-ui, sans-serif;
}

html,
body {
	background: var(--attesta-page-bg);
	color: var(--attesta-ink);
}

body,
#app {
	min-height: 100vh;
}

button,
input,
select,
textarea {
	font-family: "Space Grotesk", system-ui, sans-serif;
}

[data-slot="dialog-trigger"] {
	background: var(--attesta-accent);
	color: var(--attesta-panel);
}

input,
textarea,
[data-slot="select-trigger"],
[data-slot="checkbox"][data-state="unchecked"],
[data-slot="select-content"] {
  background: var(--attesta-panel) !important;
  border-color: var(--attesta-border) !important;
}

[data-slot="dialog-close"] {
  background: var(--attesta-danger) !important;
  color: var(--attesta-panel) !important;
}

[data-slot="field-set"] [data-slot="button"] {
  background: var(--attesta-panel) !important;
  border-color: var(--attesta-accent) !important;
  color: var(--attesta-accent) !important;
}

formata-form {
	--attesta-formata-muted: var(--attesta-muted);
	--attesta-formata-accent: var(--attesta-accent);
	--attesta-formata-panel: var(--attesta-panel);
	--attesta-formata-ink: var(--attesta-ink);
	--attesta-formata-border: var(--attesta-border);
	--attesta-formata-button-contrast: var(--attesta-panel);
}
`

const formataBuilderPreviewShadowOverrides = `
:host {
	--formata-muted: var(--attesta-formata-muted);
	--formata-accent: var(--attesta-formata-accent);
}

[data-slot="field-legend"] {
	color: var(--attesta-formata-ink) !important;
	font-family: "Space Grotesk", system-ui, sans-serif;
}

[data-slot="field-description"] {
	color: var(--attesta-formata-muted) !important;
	font-family: "Space Grotesk", system-ui, sans-serif;
	font-size: 13px;
}

[data-slot="input"]::file-selector-button,
[data-slot="button"] {
	background: var(--attesta-formata-panel) !important;
	color: var(--attesta-formata-accent) !important;
	border: 1px solid var(--attesta-formata-accent) !important;
	cursor: pointer;
	border-radius: 4px;
}

[data-slot="slider-range"] {
	background: var(--attesta-formata-accent) !important;
}

[data-slot="slider-track"] {
	background: var(--attesta-formata-panel) !important;
}

[data-slot="input"],
[data-slot="select-trigger"],
[data-slot="select-content"],
[data-slot="checkbox"],
[data-slot="radio-group-item"],
input,
select,
textarea {
	background: var(--attesta-formata-panel) !important;
	border-color: var(--attesta-formata-border) !important;
	color: var(--attesta-formata-ink) !important;
	font-family: "Space Grotesk", system-ui, sans-serif !important;
}

button[type="submit"] {
	display: none;
}
`

var formataBuilderOverrideSnippet = []byte(fmt.Sprintf(
	`<style data-attesta-formata-builder-overrides>%s</style><script>(function(){const selector="formata-form";const styleContent=%q;const styleSelector="style[data-attesta-formata-builder-overrides]";const apply=(component,attempt=0)=>{if(!(component instanceof HTMLElement)){return}const shadowRoot=component.shadowRoot;if(!shadowRoot){if(attempt<10){window.requestAnimationFrame(()=>apply(component,attempt+1))}return}let style=shadowRoot.querySelector(styleSelector);if(!(style instanceof HTMLStyleElement)){style=document.createElement("style");style.dataset.attestaFormataBuilderOverrides="true";shadowRoot.appendChild(style)}style.textContent=styleContent};const scan=()=>{document.querySelectorAll(selector).forEach((component)=>apply(component))};if(document.readyState==="loading"){document.addEventListener("DOMContentLoaded",scan,{once:true})}else{scan()}new MutationObserver(scan).observe(document.documentElement,{childList:true,subtree:true})})();</script>`,
	formataBuilderChromeOverrides,
	formataBuilderPreviewShadowOverrides,
))

// formataBuilderAssets contains the Formata builder frontend.
//
//go:embed formata-arch/attesta/index.html formata-arch/attesta/assets/* formata-arch/public/*
var formataBuilderAssets embed.FS

var formataBuilderRoots = []string{
	"formata-arch/attesta",
	"formata-arch/public",
}

func formataBuilderStreamMaxBytes() int64 {
	const defaultMax = int64(1 << 20)
	value := int64(intEnvOr("FORMATA_STREAM_MAX_BYTES", int(defaultMax)))
	if value <= 0 {
		return defaultMax
	}
	return value
}

func (s *Server) handleOrgAdminFormataBuilder(w http.ResponseWriter, r *http.Request) {
	pathValue := strings.TrimSpace(r.URL.Path)
	isRootPath := pathValue == "/org-admin/formata-builder" || pathValue == "/org-admin/formata-builder/"
	streamPath, isStreamPath := strings.CutPrefix(pathValue, "/org-admin/formata-builder/stream/")

	switch r.Method {
	case http.MethodGet:
		user, _, ok := s.requireAuthenticatedPage(w, r)
		if !ok {
			return
		}
		allowed, err := s.canViewFormataBuilder(r.Context(), user)
		if err != nil {
			logAndHTTPError(w, r, http.StatusBadGateway, "cerbos check failed", err, "cerbos check failed for formata builder view")
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if isStreamPath {
			s.handleOrgAdminFormataBuilderStream(w, r, strings.TrimSpace(streamPath), user)
			return
		}
		s.serveEmbeddedFormataBuilder(w, r, isRootPath)
		return
	case http.MethodPost:
		if !isRootPath {
			http.NotFound(w, r)
			return
		}
		user, _, ok := s.requireAuthenticatedPost(w, r)
		if !ok {
			return
		}
		allowed, err := s.canSaveFormataBuilder(r.Context(), user)
		if err != nil {
			logAndHTTPError(w, r, http.StatusBadGateway, "cerbos check failed", err, "cerbos check failed for formata builder save")
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, formataBuilderStreamMaxBytes())
		body, err := io.ReadAll(r.Body)
		if err != nil {
			if isRequestTooLarge(err) {
				http.Error(w, "stream body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		stream := strings.TrimSpace(string(body))
		if stream == "" {
			http.Error(w, "stream is required", http.StatusBadRequest)
			return
		}
		if s.store == nil {
			http.Error(w, "store not configured", http.StatusInternalServerError)
			return
		}

		streamIDValue := strings.TrimSpace(r.URL.Query().Get("stream"))
		createNew := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("new")), "true")
		if streamIDValue == "" || createNew {
			if _, err := s.store.SaveFormataBuilderStream(r.Context(), FormataBuilderStream{
				Stream:          stream,
				UpdatedAt:       s.nowUTC(),
				CreatedByUserID: formataStreamUserID(user),
				UpdatedByUserID: formataStreamUserID(user),
			}); err != nil {
				http.Error(w, "failed to save stream", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		streamID, err := primitive.ObjectIDFromHex(streamIDValue)
		if err != nil {
			http.Error(w, "invalid stream id", http.StatusBadRequest)
			return
		}
		existing, err := s.store.LoadFormataBuilderStreamByID(r.Context(), streamID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "stream not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to load stream", http.StatusInternalServerError)
			return
		}
		hasProcesses, err := s.store.HasProcessesByWorkflow(r.Context(), streamID.Hex())
		if err != nil {
			http.Error(w, "failed to check stream instances", http.StatusInternalServerError)
			return
		}
		editable, requiresPurge, err := s.formataBuilderStreamEditState(r.Context(), user, *existing)
		if err != nil {
			http.Error(w, "failed to check stream instances", http.StatusInternalServerError)
			return
		}
		if !editable {
			if strings.TrimSpace(formataStreamCreatorID(*existing)) == strings.TrimSpace(formataStreamUserID(user)) && hasProcesses {
				http.Error(w, "stream is no longer editable", http.StatusConflict)
				return
			}
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if requiresPurge {
			if err := s.store.DeleteWorkflowData(r.Context(), streamID.Hex()); err != nil {
				http.Error(w, "failed to delete stream data", http.StatusInternalServerError)
				return
			}
		}
		if _, err := s.store.UpdateFormataBuilderStream(r.Context(), FormataBuilderStream{
			ID:              existing.ID,
			Stream:          stream,
			UpdatedAt:       s.nowUTC(),
			CreatedByUserID: formataStreamCreatorID(*existing),
			UpdatedByUserID: formataStreamUserID(user),
		}); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "stream not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to save stream", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleOrgAdminFormataBuilderStream(w http.ResponseWriter, r *http.Request, streamIDValue string, user *AccountUser) {
	if s.store == nil {
		http.Error(w, "store not configured", http.StatusInternalServerError)
		return
	}
	streamID, err := primitive.ObjectIDFromHex(streamIDValue)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	stream, err := s.store.LoadFormataBuilderStreamByID(r.Context(), streamID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load stream", http.StatusInternalServerError)
		return
	}
	payload, err := formataBuilderStreamJSON(stream.Stream)
	if err != nil {
		http.Error(w, "failed to parse stream", http.StatusInternalServerError)
		return
	}
	root, ok := payload.(map[string]interface{})
	if !ok {
		root = map[string]interface{}{"stream": payload}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(root); err != nil {
		http.Error(w, "failed to encode stream", http.StatusInternalServerError)
	}
}

func (s *Server) formataBuilderStreamEditState(ctx context.Context, user *AccountUser, stream FormataBuilderStream) (bool, bool, error) {
	if s.store == nil || user == nil || stream.ID.IsZero() {
		return false, false, nil
	}
	hasProcesses, err := s.store.HasProcessesByWorkflow(ctx, stream.ID.Hex())
	if err != nil {
		return false, false, err
	}
	editable, err := s.canEditStream(ctx, user, stream.ID.Hex(), formataStreamCreatorID(stream), hasProcesses)
	if err != nil {
		return false, false, err
	}
	return editable, editable && hasProcesses, nil
}

func (s *Server) serveEmbeddedFormataBuilder(w http.ResponseWriter, r *http.Request, isRootPath bool) {
	relativePath := strings.TrimPrefix(strings.TrimSpace(r.URL.Path), "/org-admin/formata-builder")
	relativePath = strings.TrimPrefix(relativePath, "/")
	if isRootPath || relativePath == "" {
		relativePath = "index.html"
	}
	cleaned := strings.TrimPrefix(path.Clean("/"+relativePath), "/")
	if cleaned == "" || cleaned == "." {
		cleaned = "index.html"
	}

	data, contentType, err := readFormataBuilderAsset(cleaned)
	if err != nil && path.Ext(cleaned) == "" {
		data, contentType, err = readFormataBuilderAsset("index.html")
	}
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load formata builder", http.StatusInternalServerError)
		return
	}
	// Avoid stale cached bundles because we rewrite absolute asset paths at serve time.
	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if shouldRewriteFormataAssetContent(cleaned, contentType) {
		data = bytes.ReplaceAll(data, []byte("/formata-arch/"), []byte("/org-admin/formata-builder/"))
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/html") || strings.EqualFold(path.Ext(cleaned), ".html") {
			data = injectFormataBuilderOverrides(data)
		}
	}
	_, _ = w.Write(data)
}

func injectFormataBuilderOverrides(data []byte) []byte {
	if bytes.Contains(data, []byte("data-attesta-formata-builder-overrides")) {
		return data
	}
	return bytes.Replace(data, []byte("</head>"), append(formataBuilderOverrideSnippet, []byte("</head>")...), 1)
}

func readFormataBuilderAsset(relativePath string) ([]byte, string, error) {
	for _, root := range formataBuilderRoots {
		candidate := path.Clean(path.Join(root, relativePath))
		if !strings.HasPrefix(candidate, root) {
			continue
		}
		data, err := fs.ReadFile(formataBuilderAssets, candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, "", err
		}
		contentType := mime.TypeByExtension(path.Ext(candidate))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		return data, contentType, nil
	}
	return nil, "", fs.ErrNotExist
}

func shouldRewriteFormataAssetContent(assetPath, contentType string) bool {
	lowerType := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(lowerType, "text/html"):
		return true
	case strings.Contains(lowerType, "javascript"):
		return true
	case strings.Contains(lowerType, "css"):
		return true
	}
	switch strings.ToLower(path.Ext(strings.TrimSpace(assetPath))) {
	case ".html", ".js", ".css":
		return true
	}
	return false
}

func formataBuilderStreamJSON(stream string) (interface{}, error) {
	var payload interface{}
	if err := yaml.Unmarshal([]byte(stream), &payload); err != nil {
		return nil, err
	}
	return normalizeYAMLJSONValue(payload), nil
}

func normalizeYAMLJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeYAMLJSONValue(nested)
		}
		return normalized
	case map[interface{}]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			normalized[fmt.Sprint(key)] = normalizeYAMLJSONValue(nested)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeYAMLJSONValue(item))
		}
		return normalized
	default:
		return value
	}
}
