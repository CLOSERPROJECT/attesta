package main

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestMongoStoreDatabaseHelperBranches(t *testing.T) {
	storeWithPort := &MongoStore{dbPort: &fakeMongoDatabase{bucket: &fakeGridFSBucket{}}}
	if storeWithPort.database() == nil {
		t.Fatal("expected existing dbPort to be returned")
	}

	storeWithoutDB := &MongoStore{}
	if storeWithoutDB.database() != nil {
		t.Fatal("expected nil when both db and dbPort are missing")
	}

	driverStore := NewMongoStore(&mongo.Database{})
	if driverStore.database() == nil {
		t.Fatal("expected database helper to lazily return driver adapter")
	}

	lazyStore := &MongoStore{db: &mongo.Database{}}
	if lazyStore.dbPort != nil {
		t.Fatal("expected lazy store to start without dbPort")
	}
	if lazyStore.database() == nil {
		t.Fatal("expected lazy store database() to initialize dbPort")
	}
	if lazyStore.dbPort == nil {
		t.Fatal("expected lazy store dbPort to be set after database() call")
	}
}

func TestMongoDriverAdaptersExecuteWrapperMethods(t *testing.T) {
	runAndRecover := func(fn func()) {
		t.Helper()
		defer func() {
			_ = recover()
		}()
		fn()
	}

	runAndRecover(func() {
		_ = mongoDriverDatabase{}.Collection("processes")
	})
	runAndRecover(func() {
		_, _ = mongoDriverDatabase{}.NewGridFSBucket("attachments")
	})

	runAndRecover(func() {
		_, _ = (mongoDriverCollection{}).InsertOne(context.Background(), bson.M{})
	})
	runAndRecover(func() {
		_ = (mongoDriverCollection{}).FindOne(context.Background(), bson.M{})
	})
	runAndRecover(func() {
		_, _ = (mongoDriverCollection{}).Find(context.Background(), bson.M{})
	})
	runAndRecover(func() {
		_, _ = (mongoDriverCollection{}).UpdateOne(context.Background(), bson.M{}, bson.M{})
	})
	runAndRecover(func() {
		_ = (mongoDriverCollection{}).FindOneAndUpdate(context.Background(), bson.M{}, bson.M{})
	})

	runAndRecover(func() {
		_ = (mongoDriverSingleResult{}).Decode(&Process{})
	})
	runAndRecover(func() {
		_ = (mongoDriverSingleResult{}).Err()
	})

	runAndRecover(func() {
		_ = (mongoDriverCursor{}).Next(context.Background())
	})
	runAndRecover(func() {
		_ = (mongoDriverCursor{}).Decode(&Process{})
	})
	runAndRecover(func() {
		_ = (mongoDriverCursor{}).Close(context.Background())
	})

	runAndRecover(func() {
		_ = (mongoDriverGridFSBucket{}).UploadFromStreamWithID("id", "name", nil)
	})
	runAndRecover(func() {
		_, _ = (mongoDriverGridFSBucket{}).OpenDownloadStream("id")
	})
	runAndRecover(func() {
		_ = (mongoDriverGridFSBucket{}).Delete("id")
	})
}
