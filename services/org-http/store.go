package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type artifactMeta struct {
	OrgID       string    `json:"orgId,omitempty"`
	ContentType string    `json:"contentType"`
	Filename    string    `json:"filename,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type diskStore struct {
	root string
}

func newDiskStore(root string) (*diskStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("data dir is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &diskStore{root: root}, nil
}

func (s *diskStore) put(orgID, contentType, filename string, body []byte) (id string, err error) {
	id, err = newArtifactID()
	if err != nil {
		return "", err
	}
	meta := artifactMeta{
		OrgID:       strings.TrimSpace(orgID),
		ContentType: contentType,
		Filename:    filename,
		CreatedAt:   time.Now().UTC(),
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	dataPath := s.dataPath(id)
	metaPath := s.metaPath(id)
	if err := os.WriteFile(dataPath, body, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		_ = os.Remove(dataPath)
		return "", err
	}
	return id, nil
}

func (s *diskStore) get(id string) (meta artifactMeta, data []byte, err error) {
	id = strings.TrimSpace(id)
	if !validArtifactID(id) {
		return artifactMeta{}, nil, os.ErrNotExist
	}
	metaBytes, err := os.ReadFile(s.metaPath(id))
	if err != nil {
		return artifactMeta{}, nil, err
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return artifactMeta{}, nil, err
	}
	data, err = os.ReadFile(s.dataPath(id))
	if err != nil {
		return artifactMeta{}, nil, err
	}
	return meta, data, nil
}

func (s *diskStore) dataPath(id string) string {
	return filepath.Join(s.root, id+".bin")
}

func (s *diskStore) metaPath(id string) string {
	return filepath.Join(s.root, id+".meta.json")
}

func newArtifactID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
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
