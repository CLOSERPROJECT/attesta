package main

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
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
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
