package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoStoreSubstepOverrideRoundTripUsesEncodedField(t *testing.T) {
	processID := primitive.NewObjectID()
	createdAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	collection := &fakeMongoCollection{
		findOneFn: func(_ context.Context, filter interface{}, _ ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				process := v.(*Process)
				*process = Process{
					ID: processID,
					Overrides: map[string]SubstepOverride{
						"1_1": {
							SubstepID: "1.1",
							Schema:    map[string]interface{}{"type": "object"},
							Reason:    "existing",
							CreatedAt: createdAt,
						},
					},
				}
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"processes": collection}}
	store := &MongoStore{dbPort: db}

	override, err := store.GetSubstepOverride(t.Context(), processID, "1.1")
	if err != nil {
		t.Fatalf("GetSubstepOverride: %v", err)
	}
	if override.SubstepID != "1.1" || override.Reason != "existing" {
		t.Fatalf("override = %#v", override)
	}

	err = store.SaveSubstepOverride(t.Context(), processID, "wf", "1.1", SubstepOverride{
		SubstepID: "1.1",
		Schema:    map[string]interface{}{"type": "object", "title": "Updated"},
		Reason:    "updated",
		CreatedAt: time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SaveSubstepOverride: %v", err)
	}
	if len(collection.findOneAndUpdUpdate) != 1 {
		t.Fatalf("expected one update, got %d", len(collection.findOneAndUpdUpdate))
	}
	updateDoc := collection.findOneAndUpdUpdate[0].(bson.M)
	setDoc := updateDoc["$set"].(bson.M)
	if _, ok := setDoc["substepOverrides.1_1"]; !ok {
		t.Fatalf("missing encoded override key in update: %#v", setDoc)
	}
	saved := setDoc["substepOverrides.1_1"].(SubstepOverride)
	if !saved.CreatedAt.Equal(createdAt) {
		t.Fatalf("createdAt = %s, want %s", saved.CreatedAt, createdAt)
	}
	if !reflect.DeepEqual(collection.findOneAndUpdFilter[0], bson.M{"_id": processID}) {
		t.Fatalf("filter = %#v", collection.findOneAndUpdFilter[0])
	}
}
