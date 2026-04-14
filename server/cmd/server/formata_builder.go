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
	}
	_, _ = w.Write(data)
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
