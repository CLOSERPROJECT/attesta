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

func TestMongoStoreSaveFormataBuilderStream(t *testing.T) {
	updatedAt := time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)
	creatorID := "creator-1"
	userID := "updater-1"

	streamCollection := &fakeMongoCollection{}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{
			collectionFormataStream: streamCollection,
		},
	}
	store := &MongoStore{dbPort: db}

	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		CreatedByUserID: creatorID,
		Stream:          "stream-v1",
		UpdatedAt:       updatedAt,
		UpdatedByUserID: userID,
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream error: %v", err)
	}
	if saved.Stream != "stream-v1" {
		t.Fatalf("saved stream = %q, want %q", saved.Stream, "stream-v1")
	}
	if len(streamCollection.insertDocuments) != 1 {
		t.Fatalf("expected one insert call, got %d", len(streamCollection.insertDocuments))
	}
	inserted, ok := streamCollection.insertDocuments[0].(FormataBuilderStream)
	if !ok {
		t.Fatalf("insert document type = %T, want FormataBuilderStream", streamCollection.insertDocuments[0])
	}
	if inserted.ID.IsZero() {
		t.Fatal("expected non-zero inserted stream id")
	}
	if inserted.Stream != "stream-v1" {
		t.Fatalf("inserted stream = %q, want %q", inserted.Stream, "stream-v1")
	}
	if inserted.CreatedByUserID != creatorID {
		t.Fatalf("inserted createdByUserID = %q, want %q", inserted.CreatedByUserID, creatorID)
	}
	if inserted.UpdatedByUserID != userID {
		t.Fatalf("inserted updatedByUserID = %q, want %q", inserted.UpdatedByUserID, userID)
	}
}

func TestMongoStoreSaveFormataBuilderStreamErrors(t *testing.T) {
	t.Run("update failure", func(t *testing.T) {
		insertErr := errors.New("insert failed")
		collection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
		store := &MongoStore{dbPort: db}
		if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: "x"}); !errors.Is(err, insertErr) {
			t.Fatalf("SaveFormataBuilderStream error = %v, want %v", err, insertErr)
		}
	})
}

func TestMongoStoreUpdateFormataBuilderStream(t *testing.T) {
	streamID := primitive.NewObjectID()
	updatedAt := time.Date(2026, 3, 6, 10, 30, 0, 0, time.UTC)
	collection := &fakeMongoCollection{
		updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	saved, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{
		ID:              streamID,
		Stream:          "stream-v2",
		UpdatedAt:       updatedAt,
		CreatedByUserID: "creator-2",
		UpdatedByUserID: "updater-2",
	})
	if err != nil {
		t.Fatalf("UpdateFormataBuilderStream error: %v", err)
	}
	if saved.ID != streamID {
		t.Fatalf("saved id = %s, want %s", saved.ID.Hex(), streamID.Hex())
	}
	if len(collection.updateOneFilters) != 1 || !reflect.DeepEqual(collection.updateOneFilters[0], bson.M{"_id": streamID}) {
		t.Fatalf("updateOne filter = %#v, want stream id filter", collection.updateOneFilters)
	}
	if len(collection.updateOneUpdates) != 1 {
		t.Fatalf("expected one update call, got %d", len(collection.updateOneUpdates))
	}
	updateDoc, ok := collection.updateOneUpdates[0].(bson.M)
	if !ok {
		t.Fatalf("update document type = %T, want bson.M", collection.updateOneUpdates[0])
	}
	setDoc, ok := updateDoc["$set"].(bson.M)
	if !ok {
		t.Fatalf("update $set type = %T, want bson.M", updateDoc["$set"])
	}
	if setDoc["stream"] != "stream-v2" {
		t.Fatalf("updated stream = %#v, want %q", setDoc["stream"], "stream-v2")
	}
}

func TestMongoStoreUpdateFormataBuilderStreamErrors(t *testing.T) {
	t.Run("update failure", func(t *testing.T) {
		updateErr := errors.New("update failed")
		collection := &fakeMongoCollection{
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, updateErr
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
		store := &MongoStore{dbPort: db}
		if _, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{ID: primitive.NewObjectID(), Stream: "x"}); !errors.Is(err, updateErr) {
			t.Fatalf("UpdateFormataBuilderStream error = %v, want %v", err, updateErr)
		}
	})

	t.Run("missing stream id", func(t *testing.T) {
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		if _, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: "x"}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("UpdateFormataBuilderStream error = %v, want mongo.ErrNoDocuments", err)
		}
	})

	t.Run("stream not found", func(t *testing.T) {
		collection := &fakeMongoCollection{
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return &mongo.UpdateResult{}, nil
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
		store := &MongoStore{dbPort: db}
		if _, err := store.UpdateFormataBuilderStream(t.Context(), FormataBuilderStream{ID: primitive.NewObjectID(), Stream: "x"}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("UpdateFormataBuilderStream error = %v, want mongo.ErrNoDocuments", err)
		}
	})
}

func TestMongoStoreLoadFormataBuilderStream(t *testing.T) {
	want := FormataBuilderStream{
		ID:              primitive.NewObjectID(),
		Stream:          "stream-v2",
		UpdatedAt:       time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
		CreatedByUserID: "creator-2",
		UpdatedByUserID: "updater-2",
	}
	collection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*FormataBuilderStream)) = want
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.LoadFormataBuilderStream(t.Context())
	if err != nil {
		t.Fatalf("LoadFormataBuilderStream error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("stream = %#v, want %#v", *got, want)
	}
	if len(collection.findOneFilters) != 1 || !reflect.DeepEqual(collection.findOneFilters[0], bson.M{}) {
		t.Fatalf("findOne filter = %#v, want empty filter", collection.findOneFilters)
	}
	if len(collection.findOneOptionsCalls) != 1 || len(collection.findOneOptionsCalls[0]) != 1 {
		t.Fatalf("expected one findOne option call, got %#v", collection.findOneOptionsCalls)
	}
	if collection.findOneOptionsCalls[0][0].Sort == nil {
		t.Fatal("expected findOne sort options")
	}
}

func TestMongoStoreListFormataBuilderStreams(t *testing.T) {
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	want := []FormataBuilderStream{
		{
			ID:        primitive.NewObjectID(),
			Stream:    "workflow-a",
			UpdatedAt: now,
		},
		{
			ID:        primitive.NewObjectID(),
			Stream:    "workflow-b",
			UpdatedAt: now.Add(-time.Minute),
		},
	}
	cursor := &fakeAnyCursor{items: []interface{}{want[0], want[1]}}
	collection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return cursor, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("streams = %#v, want %#v", got, want)
	}
	if len(collection.findFilters) != 1 || !reflect.DeepEqual(collection.findFilters[0], bson.M{}) {
		t.Fatalf("find filter = %#v, want empty filter", collection.findFilters)
	}
	if len(collection.findOptionsCalls) != 1 || len(collection.findOptionsCalls[0]) != 1 {
		t.Fatalf("find options calls = %#v, want one call with one option", collection.findOptionsCalls)
	}
	opts := collection.findOptionsCalls[0][0]
	if opts.Sort == nil {
		t.Fatal("expected find sort options")
	}
}

func TestMongoStoreLoadFormataBuilderStreamByID(t *testing.T) {
	want := FormataBuilderStream{
		ID:              primitive.NewObjectID(),
		Stream:          "stream-v3",
		CreatedByUserID: "creator-3",
	}
	collection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*FormataBuilderStream)) = want
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.LoadFormataBuilderStreamByID(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("LoadFormataBuilderStreamByID error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("stream = %#v, want %#v", *got, want)
	}
	if len(collection.findOneFilters) != 1 || !reflect.DeepEqual(collection.findOneFilters[0], bson.M{"_id": want.ID}) {
		t.Fatalf("findOne filter = %#v, want stream id filter", collection.findOneFilters)
	}
}

func TestMongoStoreDeleteFormataBuilderStream(t *testing.T) {
	streamID := primitive.NewObjectID()
	collection := &fakeMongoCollection{
		deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	if err := store.DeleteFormataBuilderStream(t.Context(), streamID); err != nil {
		t.Fatalf("DeleteFormataBuilderStream error: %v", err)
	}
	if len(collection.deleteOneFilters) != 1 || !reflect.DeepEqual(collection.deleteOneFilters[0], bson.M{"_id": streamID}) {
		t.Fatalf("deleteOne filter = %#v, want stream id filter", collection.deleteOneFilters)
	}
}

func TestMongoStoreHasProcessesByWorkflow(t *testing.T) {
	collection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	hasProcesses, err := store.HasProcessesByWorkflow(t.Context(), "workflow-1")
	if err != nil {
		t.Fatalf("HasProcessesByWorkflow error: %v", err)
	}
	if !hasProcesses {
		t.Fatal("expected workflow to have processes")
	}
	if len(collection.findOneFilters) != 1 || !reflect.DeepEqual(collection.findOneFilters[0], bson.M{"workflowKey": "workflow-1"}) {
		t.Fatalf("findOne filter = %#v, want workflow filter", collection.findOneFilters)
	}
}

func TestMongoStoreDeleteWorkflowData(t *testing.T) {
	processID := primitive.NewObjectID()
	attachmentID := primitive.NewObjectID()
	processesCollection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{bson.M{"_id": processID}}}, nil
		},
	}
	attachmentsFilesCollection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{bson.M{"_id": attachmentID}}}, nil
		},
	}
	attachmentsChunksCollection := &fakeMongoCollection{}
	notarizationsCollection := &fakeMongoCollection{}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{
			"processes":          processesCollection,
			"attachments.files":  attachmentsFilesCollection,
			"attachments.chunks": attachmentsChunksCollection,
			"notarizations":      notarizationsCollection,
		},
	}
	store := &MongoStore{dbPort: db}

	if err := store.DeleteWorkflowData(t.Context(), "workflow-1"); err != nil {
		t.Fatalf("DeleteWorkflowData error: %v", err)
	}
	if len(processesCollection.findFilters) != 1 || !reflect.DeepEqual(processesCollection.findFilters[0], bson.M{"workflowKey": "workflow-1"}) {
		t.Fatalf("process find filter = %#v, want workflow filter", processesCollection.findFilters)
	}
	if len(attachmentsFilesCollection.findFilters) != 1 || !reflect.DeepEqual(attachmentsFilesCollection.findFilters[0], bson.M{"metadata.processId": bson.M{"$in": []primitive.ObjectID{processID}}}) {
		t.Fatalf("attachment find filter = %#v", attachmentsFilesCollection.findFilters)
	}
	if len(attachmentsChunksCollection.deleteManyFilters) != 1 || !reflect.DeepEqual(attachmentsChunksCollection.deleteManyFilters[0], bson.M{"files_id": bson.M{"$in": []primitive.ObjectID{attachmentID}}}) {
		t.Fatalf("chunks delete filter = %#v", attachmentsChunksCollection.deleteManyFilters)
	}
	if len(attachmentsFilesCollection.deleteManyFilters) != 1 || !reflect.DeepEqual(attachmentsFilesCollection.deleteManyFilters[0], bson.M{"_id": bson.M{"$in": []primitive.ObjectID{attachmentID}}}) {
		t.Fatalf("files delete filter = %#v", attachmentsFilesCollection.deleteManyFilters)
	}
	if len(notarizationsCollection.deleteManyFilters) != 1 || !reflect.DeepEqual(notarizationsCollection.deleteManyFilters[0], bson.M{"processId": bson.M{"$in": []primitive.ObjectID{processID}}}) {
		t.Fatalf("notarizations delete filter = %#v", notarizationsCollection.deleteManyFilters)
	}
	if len(processesCollection.deleteManyFilters) != 1 || !reflect.DeepEqual(processesCollection.deleteManyFilters[0], bson.M{"_id": bson.M{"$in": []primitive.ObjectID{processID}}}) {
		t.Fatalf("process delete filter = %#v", processesCollection.deleteManyFilters)
	}
}

func TestMongoStoreListFormataBuilderStreamsFindError(t *testing.T) {
	findErr := errors.New("find failed")
	collection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return nil, findErr
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	if _, err := store.ListFormataBuilderStreams(t.Context()); !errors.Is(err, findErr) {
		t.Fatalf("ListFormataBuilderStreams error = %v, want %v", err, findErr)
	}
}

func TestMongoStoreListFormataBuilderStreamsSkipsDecodeErrors(t *testing.T) {
	good := FormataBuilderStream{
		ID:        primitive.NewObjectID(),
		Stream:    "workflow-a",
		UpdatedAt: time.Date(2026, 3, 6, 13, 0, 0, 0, time.UTC),
	}
	cursor := &fakeAnyCursor{
		items:       []interface{}{"bad-item", good},
		decodeErrAt: map[int]error{0: errors.New("decode failed")},
	}
	collection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return cursor, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
	store := &MongoStore{dbPort: db}

	got, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams error: %v", err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], good) {
		t.Fatalf("streams = %#v, want %#v", got, []FormataBuilderStream{good})
	}
}
