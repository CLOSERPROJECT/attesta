package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	token := strings.TrimSpace(os.Getenv("ORG_HTTP_TOKEN"))
	if token == "" {
		log.Fatal("ORG_HTTP_TOKEN is required")
	}
	dataDir := strings.TrimSpace(os.Getenv("ORG_HTTP_DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data"
	}
	addr := strings.TrimSpace(os.Getenv("ORG_HTTP_ADDR"))
	if addr == "" {
		addr = ":8090"
	}

	store, err := newDiskStore(dataDir)
	if err != nil {
		log.Fatalf("data dir: %v", err)
	}
	srv := &server{store: store, token: token}
	log.Printf("org-http listening on %s data=%s", addr, dataDir)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatal(err)
	}
}