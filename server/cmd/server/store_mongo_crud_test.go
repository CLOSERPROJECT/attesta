package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoStoreInsertProcess(t *testing.T) {
	insertedID := primitive.NewObjectID()
	collection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: insertedID}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	gotID, err := store.InsertProcess(t.Context(), Process{Status: "active"})
	if err != nil {
		t.Fatalf("InsertProcess returned error: %v", err)
	}
	if gotID != insertedID {
		t.Fatalf("InsertProcess id = %s, want %s", gotID.Hex(), insertedID.Hex())
	}
	if len(collection.insertDocuments) != 1 {
		t.Fatalf("expected one InsertOne call, got %d", len(collection.insertDocuments))
	}
}

func TestMongoStoreInsertProcessErrors(t *testing.T) {
	insertErr := errors.New("insert failed")
	collection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return nil, insertErr
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	if _, err := store.InsertProcess(t.Context(), Process{}); !errors.Is(err, insertErr) {
		t.Fatalf("InsertProcess error = %v, want %v", err, insertErr)
	}

	collection.insertOneFn = func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
		return &mongo.InsertOneResult{InsertedID: "not-an-object-id"}, nil
	}
	if _, err := store.InsertProcess(t.Context(), Process{}); err == nil {
		t.Fatal("expected invalid inserted id error")
	}
}

func TestMongoStoreLoadProcessByID(t *testing.T) {
	wantID := primitive.NewObjectID()
	want := Process{ID: wantID, Status: "active", Progress: map[string]ProcessStep{"1_1": {State: "done"}}}
	collection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Process)) = want
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.LoadProcessByID(t.Context(), wantID)
	if err != nil {
		t.Fatalf("LoadProcessByID returned error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("LoadProcessByID result = %#v, want %#v", *got, want)
	}

	loadErr := errors.New("load failed")
	collection.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
		return fakeSingleResult{err: loadErr}
	}
	if _, err := store.LoadProcessByID(t.Context(), wantID); !errors.Is(err, loadErr) {
		t.Fatalf("LoadProcessByID error = %v, want %v", err, loadErr)
	}
}

func TestMongoStoreLoadLatestProcessByWorkflow(t *testing.T) {
	want := Process{ID: primitive.NewObjectID(), CreatedAt: time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC)}
	collection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Process)) = want
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.LoadLatestProcessByWorkflow(t.Context(), "wf-a")
	if err != nil {
		t.Fatalf("LoadLatestProcessByWorkflow returned error: %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("LoadLatestProcessByWorkflow id = %s, want %s", got.ID.Hex(), want.ID.Hex())
	}
	if len(collection.findOneFilters) != 1 {
		t.Fatalf("expected one findOne filter, got %d", len(collection.findOneFilters))
	}
	if !reflect.DeepEqual(collection.findOneFilters[0], bson.M{"workflowKey": "wf-a"}) {
		t.Fatalf("findOne filter = %#v, want workflow filter", collection.findOneFilters[0])
	}
	if len(collection.findOneOptionsCalls) != 1 || len(collection.findOneOptionsCalls[0]) != 1 {
		t.Fatalf("expected one findOne options call, got %#v", collection.findOneOptionsCalls)
	}
	sortDoc, ok := collection.findOneOptionsCalls[0][0].Sort.(bson.D)
	if !ok {
		t.Fatalf("expected bson.D sort option, got %#v", collection.findOneOptionsCalls[0][0].Sort)
	}
	wantSort := bson.D{{Key: "createdAt", Value: -1}}
	if !reflect.DeepEqual(sortDoc, wantSort) {
		t.Fatalf("sort option = %#v, want %#v", sortDoc, wantSort)
	}

	loadErr := errors.New("latest failed")
	collection.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
		return fakeSingleResult{err: loadErr}
	}
	if _, err := store.LoadLatestProcessByWorkflow(t.Context(), "wf-a"); !errors.Is(err, loadErr) {
		t.Fatalf("LoadLatestProcessByWorkflow error = %v, want %v", err, loadErr)
	}
}

func TestMongoStoreListRecentProcessesByWorkflow(t *testing.T) {
	cursor := &fakeCursor{
		docs: []Process{
			{ID: primitive.NewObjectID(), Status: "a"},
			{ID: primitive.NewObjectID(), Status: "bad"},
			{ID: primitive.NewObjectID(), Status: "b"},
		},
		decodeErrAt: map[int]error{1: errors.New("decode failed")},
	}
	collection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return cursor, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	processes, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-a", 25)
	if err != nil {
		t.Fatalf("ListRecentProcessesByWorkflow returned error: %v", err)
	}
	if len(processes) != 2 {
		t.Fatalf("expected decode failure to be skipped, got %d entries", len(processes))
	}
	if !cursor.closed {
		t.Fatal("expected cursor Close to be called")
	}
	if len(collection.findFilters) != 1 {
		t.Fatalf("expected one find filter, got %d", len(collection.findFilters))
	}
	if !reflect.DeepEqual(collection.findFilters[0], bson.M{"workflowKey": "wf-a"}) {
		t.Fatalf("find filter = %#v, want workflow filter", collection.findFilters[0])
	}

	findErr := errors.New("find failed")
	collection.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
		return nil, findErr
	}
	if _, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-a", 1); !errors.Is(err, findErr) {
		t.Fatalf("ListRecentProcessesByWorkflow error = %v, want %v", err, findErr)
	}
}

func TestMongoStoreDefaultWorkflowFallbackFilter(t *testing.T) {
	collection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeCursor{}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	if _, err := store.ListRecentProcessesByWorkflow(t.Context(), "workflow", 10); err != nil {
		t.Fatalf("ListRecentProcessesByWorkflow returned error: %v", err)
	}
	want := bson.M{"$or": []bson.M{{"workflowKey": "workflow"}, {"workflowKey": bson.M{"$exists": false}}}}
	if len(collection.findFilters) != 1 || !reflect.DeepEqual(collection.findFilters[0], want) {
		t.Fatalf("find filter = %#v, want %#v", collection.findFilters, want)
	}
}

func TestMongoStoreUpdateProcessProgress(t *testing.T) {
	collection := &fakeMongoCollection{
		findOneAndUpdateFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
			return fakeSingleResult{}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}
	id := primitive.NewObjectID()
	progress := ProcessStep{State: "done"}

	if err := store.UpdateProcessProgress(t.Context(), id, "wf-a", "1.1", progress); err != nil {
		t.Fatalf("UpdateProcessProgress returned error: %v", err)
	}
	if len(collection.findOneAndUpdFilter) != 1 || len(collection.findOneAndUpdUpdate) != 1 {
		t.Fatalf("expected one FindOneAndUpdate call, got filters=%d updates=%d", len(collection.findOneAndUpdFilter), len(collection.findOneAndUpdUpdate))
	}
	expectedUpdate := bson.M{
		"$set": bson.M{
			"workflowKey":  "wf-a",
			"progress.1_1": progress,
		},
	}
	if !reflect.DeepEqual(collection.findOneAndUpdUpdate[0], expectedUpdate) {
		t.Fatalf("update doc = %#v, want %#v", collection.findOneAndUpdUpdate[0], expectedUpdate)
	}

	updateErr := errors.New("update failed")
	collection.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
		return fakeSingleResult{err: updateErr}
	}
	if err := store.UpdateProcessProgress(t.Context(), id, "wf-a", "1.1", progress); !errors.Is(err, updateErr) {
		t.Fatalf("UpdateProcessProgress error = %v, want %v", err, updateErr)
	}
}

func TestMongoStoreUpdateProcessStatusAndInsertNotarization(t *testing.T) {
	processes := &fakeMongoCollection{}
	notarizations := &fakeMongoCollection{}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{
			"processes":     processes,
			"notarizations": notarizations,
		},
	}
	store := &MongoStore{dbPort: db}
	id := primitive.NewObjectID()

	if err := store.UpdateProcessStatus(t.Context(), id, "wf-a", "done"); err != nil {
		t.Fatalf("UpdateProcessStatus returned error: %v", err)
	}
	if len(processes.updateOneFilters) != 1 || len(processes.updateOneUpdates) != 1 {
		t.Fatalf("expected one UpdateOne call, got filters=%d updates=%d", len(processes.updateOneFilters), len(processes.updateOneUpdates))
	}
	expectedStatusUpdate := bson.M{"$set": bson.M{"status": "done", "workflowKey": "wf-a"}}
	if !reflect.DeepEqual(processes.updateOneUpdates[0], expectedStatusUpdate) {
		t.Fatalf("status update = %#v, want %#v", processes.updateOneUpdates[0], expectedStatusUpdate)
	}

	notary := Notarization{ID: primitive.NewObjectID(), ProcessID: id, SubstepID: "1.1"}
	if err := store.InsertNotarization(t.Context(), notary); err != nil {
		t.Fatalf("InsertNotarization returned error: %v", err)
	}
	if len(notarizations.insertDocuments) != 1 {
		t.Fatalf("expected one notarization insert, got %d", len(notarizations.insertDocuments))
	}

	updateErr := errors.New("status failed")
	processes.updateOneFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
		return nil, updateErr
	}
	if err := store.UpdateProcessStatus(t.Context(), id, "wf-a", "done"); !errors.Is(err, updateErr) {
		t.Fatalf("UpdateProcessStatus error = %v, want %v", err, updateErr)
	}

	insertErr := errors.New("insert failed")
	notarizations.insertOneFn = func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
		return nil, insertErr
	}
	if err := store.InsertNotarization(t.Context(), notary); !errors.Is(err, insertErr) {
		t.Fatalf("InsertNotarization error = %v, want %v", err, insertErr)
	}
}
