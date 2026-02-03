//go:build integration
// +build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestIntegrationCompleteSubstepWithMongoAndCerbos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mongoURI := envOr("MONGODB_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skipf("skip integration test: mongo unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("skip integration test: mongo ping failed: %v", err)
	}

	cerbosURL := envOr("CERBOS_URL", "http://localhost:3592")
	if !isCerbosHealthy(cerbosURL) {
		t.Skipf("skip integration test: cerbos unavailable at %s", cerbosURL)
	}

	db := client.Database("closer_demo_integration_test")
	store := &MongoStore{db: db}
	authorizer := NewCerbosAuthorizer(cerbosURL, http.DefaultClient, time.Now)

	t.Run("allowed dep1 completion", func(t *testing.T) {
		processID := seedIntegrationProcess(t, db, store)
		server := integrationServer(store, authorizer)

		req := httptest.NewRequest(http.MethodPost, "/process/"+processID.Hex()+"/substep/1.1/complete", strings.NewReader("value=10"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
		rr := httptest.NewRecorder()

		server.handleCompleteSubstep(rr, req, processID.Hex(), "1.1")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		updated, err := store.LoadProcessByID(context.Background(), processID)
		if err != nil {
			t.Fatalf("load process after completion: %v", err)
		}
		if updated.Progress["1_1"].State != "done" {
			t.Fatalf("expected 1_1 state done, got %q", updated.Progress["1_1"].State)
		}

		count, err := db.Collection("notarizations").CountDocuments(context.Background(), bson.M{"processId": processID})
		if err != nil {
			t.Fatalf("count notarizations: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 notarization, got %d", count)
		}

		cleanupIntegrationProcess(t, db, processID)
	})

	t.Run("denied dep2 completion", func(t *testing.T) {
		processID := seedIntegrationProcess(t, db, store)
		server := integrationServer(store, authorizer)

		req := httptest.NewRequest(http.MethodPost, "/process/"+processID.Hex()+"/substep/1.1/complete", strings.NewReader("value=10"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u2|dep2"})
		rr := httptest.NewRecorder()

		server.handleCompleteSubstep(rr, req, processID.Hex(), "1.1")
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
		}

		updated, err := store.LoadProcessByID(context.Background(), processID)
		if err != nil {
			t.Fatalf("load process after denied completion: %v", err)
		}
		if updated.Progress["1_1"].State != "pending" {
			t.Fatalf("expected 1_1 state pending, got %q", updated.Progress["1_1"].State)
		}

		count, err := db.Collection("notarizations").CountDocuments(context.Background(), bson.M{"processId": processID})
		if err != nil {
			t.Fatalf("count notarizations: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 notarizations on deny, got %d", count)
		}

		cleanupIntegrationProcess(t, db, processID)
	})
}

func integrationServer(store Store, authorizer Authorizer) *Server {
	return &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: authorizer,
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
}

func seedIntegrationProcess(t *testing.T, db *mongo.Database, store Store) primitive.ObjectID {
	t.Helper()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
			"1_2": {State: "pending"},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
			"3_2": {State: "pending"},
		},
	}
	if _, err := store.InsertProcess(context.Background(), process); err != nil {
		t.Fatalf("insert integration process: %v", err)
	}
	cleanupIntegrationProcess(t, db, process.ID)
	return process.ID
}

func cleanupIntegrationProcess(t *testing.T, db *mongo.Database, processID primitive.ObjectID) {
	t.Helper()
	_, _ = db.Collection("processes").DeleteOne(context.Background(), bson.M{"_id": processID})
	_, _ = db.Collection("notarizations").DeleteMany(context.Background(), bson.M{"processId": processID})
}

func isCerbosHealthy(url string) bool {
	endpoint := strings.TrimSuffix(url, "/") + "/_cerbos/health"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}
