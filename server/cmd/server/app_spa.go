package main

import (
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const appMountPath = "/app"

func appBuildDir() string {
	if dir := strings.TrimSpace(os.Getenv("ATTESTA_APP_DIR")); dir != "" {
		return dir
	}
	if root := strings.TrimSpace(os.Getenv("ATTESTA_ROOT_DIR")); root != "" {
		return filepath.Join(root, "app", "build")
	}
	return filepath.Join("..", "app", "build")
}

func (s *Server) handleAppSPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.requireVerifiedPage(w, r); !ok {
		return
	}
	if s.appDevServer != "" {
		s.proxyAppDev(w, r)
		return
	}
	s.serveAppStatic(w, r)
}

func (s *Server) proxyAppDev(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(s.appDevServer)
	if err != nil {
		http.Error(w, "invalid APP_DEV_SERVER", http.StatusBadGateway)
		return
	}
	proxy := &httputil.ReverseProxy{}
	proxy.Rewrite = func(req *httputil.ProxyRequest) {
		req.SetURL(target)
		req.Out.Host = target.Host
		if origin := req.In.Header.Get("Origin"); origin != "" {
			req.Out.Header.Set("Origin", origin)
		}
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, err error) {
		http.Error(rw, "app dev server unavailable: "+err.Error(), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func (s *Server) serveAppStatic(w http.ResponseWriter, r *http.Request) {
	relativePath := strings.TrimPrefix(strings.TrimSpace(r.URL.Path), appMountPath)
	relativePath = strings.TrimPrefix(relativePath, "/")
	if relativePath == "" {
		relativePath = "index.html"
	}
	cleaned := strings.TrimPrefix(path.Clean("/"+relativePath), "/")
	if cleaned == "" || cleaned == "." {
		cleaned = "index.html"
	}

	data, contentType, err := readAppBuildAsset(cleaned)
	if err != nil && path.Ext(cleaned) == "" {
		data, contentType, err = readAppBuildAsset("index.html")
	}
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load app", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

func readAppBuildAsset(relativePath string) ([]byte, string, error) {
	root := appBuildDir()
	full := filepath.Join(root, filepath.FromSlash(relativePath))
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	cleanFull, err := filepath.Abs(full)
	if err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(cleanRoot, cleanFull)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", fs.ErrNotExist
	}
	data, err := os.ReadFile(cleanFull)
	if err != nil {
		return nil, "", err
	}
	contentType := mime.TypeByExtension(path.Ext(relativePath))
	if contentType == "" {
		contentType = "application/octet-stream"
		if path.Ext(relativePath) == "" || strings.EqualFold(path.Ext(relativePath), ".html") {
			contentType = "text/html; charset=utf-8"
		}
	}
	return data, contentType, nil
}
