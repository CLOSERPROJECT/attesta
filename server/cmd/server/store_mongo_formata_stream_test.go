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
	userID := primitive.NewObjectID()

	streamCollection := &fakeMongoCollection{}
	streamCollection.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*FormataBuilderStream)) = FormataBuilderStream{
				ID:                   primitive.NewObjectID(),
				Key:                  formataBuilderStreamKey,
				Stream:               "stream-v1",
				UpdatedAt:            updatedAt,
				UpdatedByUserMongoID: userID,
			}
			return nil
		}}
	}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{
			collectionFormataStream: streamCollection,
		},
	}
	store := &MongoStore{dbPort: db}

	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream:               "stream-v1",
		UpdatedAt:            updatedAt,
		UpdatedByUserMongoID: userID,
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream error: %v", err)
	}
	if saved.Stream != "stream-v1" {
		t.Fatalf("saved stream = %q, want %q", saved.Stream, "stream-v1")
	}
	if len(streamCollection.updateOneFilters) != 1 {
		t.Fatalf("expected one update call, got %d", len(streamCollection.updateOneFilters))
	}
	if !reflect.DeepEqual(streamCollection.updateOneFilters[0], bson.M{"key": formataBuilderStreamKey}) {
		t.Fatalf("update filter = %#v, want default key filter", streamCollection.updateOneFilters[0])
	}
	updateDoc, ok := streamCollection.updateOneUpdates[0].(bson.M)
	if !ok {
		t.Fatalf("update doc type = %T, want bson.M", streamCollection.updateOneUpdates[0])
	}
	setDoc, _ := updateDoc["$set"].(bson.M)
	if setDoc["stream"] != "stream-v1" {
		t.Fatalf("set stream = %#v, want %q", setDoc["stream"], "stream-v1")
	}
	if len(streamCollection.updateOneOptions) != 1 || len(streamCollection.updateOneOptions[0]) != 1 {
		t.Fatalf("expected upsert option call, got %#v", streamCollection.updateOneOptions)
	}
	if streamCollection.updateOneOptions[0][0].Upsert == nil || !*streamCollection.updateOneOptions[0][0].Upsert {
		t.Fatalf("expected upsert=true option, got %#v", streamCollection.updateOneOptions[0][0].Upsert)
	}
}

func TestMongoStoreSaveFormataBuilderStreamErrors(t *testing.T) {
	t.Run("update failure", func(t *testing.T) {
		updateErr := errors.New("update failed")
		collection := &fakeMongoCollection{
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, updateErr
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
		store := &MongoStore{dbPort: db}
		if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: "x"}); !errors.Is(err, updateErr) {
			t.Fatalf("SaveFormataBuilderStream error = %v, want %v", err, updateErr)
		}
	})

	t.Run("load after update failure", func(t *testing.T) {
		loadErr := errors.New("load failed")
		collection := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: loadErr}
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionFormataStream: collection}}
		store := &MongoStore{dbPort: db}
		if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: "x"}); !errors.Is(err, loadErr) {
			t.Fatalf("SaveFormataBuilderStream error = %v, want %v", err, loadErr)
		}
	})
}

func TestMongoStoreLoadFormataBuilderStream(t *testing.T) {
	want := FormataBuilderStream{
		ID:                   primitive.NewObjectID(),
		Key:                  formataBuilderStreamKey,
		Stream:               "stream-v2",
		UpdatedAt:            time.Date(2026, 3, 6, 11, 0, 0, 0, time.UTC),
		UpdatedByUserMongoID: primitive.NewObjectID(),
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
	if len(collection.findOneFilters) != 1 || !reflect.DeepEqual(collection.findOneFilters[0], bson.M{"key": formataBuilderStreamKey}) {
		t.Fatalf("findOne filter = %#v, want default key filter", collection.findOneFilters)
	}
}

func TestMongoStoreListFormataBuilderStreams(t *testing.T) {
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	want := []FormataBuilderStream{
		{
			ID:        primitive.NewObjectID(),
			Key:       "workflow",
			Stream:    "workflow-a",
			UpdatedAt: now,
		},
		{
			ID:        primitive.NewObjectID(),
			Key:       "secondary",
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
		Key:       "workflow",
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
