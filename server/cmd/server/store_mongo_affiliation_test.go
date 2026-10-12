package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoStoreJoinRequestCRUD(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	id := primitive.NewObjectID()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	collection := &fakeMongoCollection{
		insertOneFn: func(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: id}, nil
		},
		findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				req := v.(*JoinRequest)
				*req = JoinRequest{
					ID:              id,
					RequesterUserID: "user-1",
					RequesterEmail:  "u@example.com",
					OrgSlug:         "acme",
					Status:          AffiliationStatusPending,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				return nil
			}}
		},
		updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{MatchedCount: 1}, nil
		},
		deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		},
		findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				JoinRequest{ID: id, OrgSlug: "acme", Status: AffiliationStatusPending, RequesterUserID: "user-1"},
			}}, nil
		},
	}
	store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionJoinRequests: collection,
	}}}

	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "u@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}

	loaded, err := store.LoadJoinRequestByID(ctx, id)
	if err != nil || loaded == nil || loaded.ID != id {
		t.Fatalf("LoadJoinRequestByID: %+v err=%v", loaded, err)
	}

	updated, err := store.UpdateJoinRequest(ctx, JoinRequest{
		ID:              id,
		RequesterUserID: "user-1",
		RequesterEmail:  "u@example.com",
		OrgSlug:         "acme",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
	})
	if err != nil {
		t.Fatalf("UpdateJoinRequest: %v", err)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	if err := store.DeleteJoinRequest(ctx, id); err != nil {
		t.Fatalf("DeleteJoinRequest: %v", err)
	}

	list, err := store.ListPendingJoinRequestsByOrg(ctx, "acme")
	if err != nil || len(list) != 1 {
		t.Fatalf("ListPendingJoinRequestsByOrg: %v len=%d", err, len(list))
	}

	found, err := store.FindPendingJoinRequestByUser(ctx, "user-1")
	if err != nil || found == nil || found.ID != id {
		t.Fatalf("FindPendingJoinRequestByUser: %+v err=%v", found, err)
	}
}

func TestMongoStoreJoinRequestErrorPaths(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	boom := errors.New("mongo boom")

	t.Run("insert error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			insertOneFn: func(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.InsertJoinRequest(ctx, JoinRequest{OrgSlug: "acme"}); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: boom}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.LoadJoinRequestByID(ctx, primitive.NewObjectID()); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update zero id", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		if _, err := store.UpdateJoinRequest(ctx, JoinRequest{}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.UpdateJoinRequest(ctx, JoinRequest{ID: primitive.NewObjectID()}); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update unmatched", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return &mongo.UpdateResult{MatchedCount: 0}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.UpdateJoinRequest(ctx, JoinRequest{ID: primitive.NewObjectID()}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete zero id", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		if err := store.DeleteJoinRequest(ctx, primitive.NilObjectID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if err := store.DeleteJoinRequest(ctx, primitive.NewObjectID()); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete unmatched", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{DeletedCount: 0}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if err := store.DeleteJoinRequest(ctx, primitive.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("list find error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.ListPendingJoinRequestsByOrg(ctx, "acme"); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("list skips decode errors", func(t *testing.T) {
		t.Parallel()
		id := primitive.NewObjectID()
		coll := &fakeMongoCollection{
			findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{
					items:       []interface{}{JoinRequest{ID: id}, JoinRequest{ID: primitive.NewObjectID()}},
					decodeErrAt: map[int]error{0: errors.New("bad doc")},
				}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		list, err := store.ListPendingJoinRequestsByOrg(ctx, "")
		if err != nil || len(list) != 1 {
			t.Fatalf("list=%v err=%v", list, err)
		}
	})

	t.Run("find by user empty", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		got, err := store.FindPendingJoinRequestByUser(ctx, "  ")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("find by user not found", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		got, err := store.FindPendingJoinRequestByUser(ctx, "user-1")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("find by user error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: boom}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
		if _, err := store.FindPendingJoinRequestByUser(ctx, "user-1"); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMongoStoreOrganizationCreationRequestCRUD(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	id := primitive.NewObjectID()
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	collection := &fakeMongoCollection{
		insertOneFn: func(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: id}, nil
		},
		findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				req := v.(*OrganizationCreationRequest)
				*req = OrganizationCreationRequest{
					ID:              id,
					RequesterUserID: "user-2",
					RequesterEmail:  "v@example.com",
					ProposedName:    "New Org",
					ProposedSlug:    "new-org",
					Status:          AffiliationStatusPending,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				return nil
			}}
		},
		updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{MatchedCount: 1}, nil
		},
		deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		},
		findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				OrganizationCreationRequest{ID: id, ProposedSlug: "new-org", Status: AffiliationStatusPending},
			}}, nil
		},
	}
	store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionOrganizationCreationRequests: collection,
	}}}

	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-2",
		RequesterEmail:  "v@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}

	loaded, err := store.LoadOrganizationCreationRequestByID(ctx, id)
	if err != nil || loaded == nil || loaded.ProposedSlug != "new-org" {
		t.Fatalf("LoadOrganizationCreationRequestByID: %+v err=%v", loaded, err)
	}

	updated, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		ID:              id,
		RequesterUserID: "user-2",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
	})
	if err != nil {
		t.Fatalf("UpdateOrganizationCreationRequest: %v", err)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt")
	}

	if err := store.DeleteOrganizationCreationRequest(ctx, id); err != nil {
		t.Fatalf("DeleteOrganizationCreationRequest: %v", err)
	}

	list, err := store.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListPendingOrganizationCreationRequests: %v len=%d", err, len(list))
	}

	found, err := store.FindPendingOrganizationCreationRequestByUser(ctx, "user-2")
	if err != nil || found == nil || found.ID != id {
		t.Fatalf("FindPendingOrganizationCreationRequestByUser: %+v err=%v", found, err)
	}
}

func TestMongoStoreOrganizationCreationRequestErrorPaths(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	boom := errors.New("mongo boom")

	t.Run("insert error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			insertOneFn: func(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{}); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: boom}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.LoadOrganizationCreationRequestByID(ctx, primitive.NewObjectID()); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update zero id", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		if _, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{ID: primitive.NewObjectID()}); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("update unmatched", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			updateOneFn: func(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return &mongo.UpdateResult{MatchedCount: 0}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{ID: primitive.NewObjectID()}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete zero id", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		if err := store.DeleteOrganizationCreationRequest(ctx, primitive.NilObjectID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if err := store.DeleteOrganizationCreationRequest(ctx, primitive.NewObjectID()); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("delete unmatched", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			deleteOneFn: func(context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{DeletedCount: 0}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if err := store.DeleteOrganizationCreationRequest(ctx, primitive.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("list find error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, boom
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.ListPendingOrganizationCreationRequests(ctx); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("list skips decode errors", func(t *testing.T) {
		t.Parallel()
		id := primitive.NewObjectID()
		coll := &fakeMongoCollection{
			findFn: func(context.Context, interface{}, ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{
					items: []interface{}{
						OrganizationCreationRequest{ID: id},
						OrganizationCreationRequest{ID: primitive.NewObjectID()},
					},
					decodeErrAt: map[int]error{0: errors.New("bad doc")},
				}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		list, err := store.ListPendingOrganizationCreationRequests(ctx)
		if err != nil || len(list) != 1 {
			t.Fatalf("list=%v err=%v", list, err)
		}
	})

	t.Run("find by user empty", func(t *testing.T) {
		t.Parallel()
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		got, err := store.FindPendingOrganizationCreationRequestByUser(ctx, "")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("find by user not found", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		got, err := store.FindPendingOrganizationCreationRequestByUser(ctx, "user-2")
		if err != nil || got != nil {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("find by user error", func(t *testing.T) {
		t.Parallel()
		coll := &fakeMongoCollection{
			findOneFn: func(context.Context, interface{}, ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: boom}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizationCreationRequests: coll}}}
		if _, err := store.FindPendingOrganizationCreationRequestByUser(ctx, "user-2"); !errors.Is(err, boom) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMongoStoreListPendingJoinRequestsFilterIncludesOrg(t *testing.T) {
	t.Parallel()
	var gotFilter interface{}
	coll := &fakeMongoCollection{
		findFn: func(_ context.Context, filter interface{}, _ ...*options.FindOptions) (mongoCursorPort, error) {
			gotFilter = filter
			return &fakeAnyCursor{}, nil
		},
	}
	store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionJoinRequests: coll}}}
	if _, err := store.ListPendingJoinRequestsByOrg(t.Context(), " acme "); err != nil {
		t.Fatalf("list: %v", err)
	}
	m, ok := gotFilter.(bson.M)
	if !ok || m["orgSlug"] != "acme" || m["status"] != AffiliationStatusPending {
		t.Fatalf("filter = %#v", gotFilter)
	}
}
