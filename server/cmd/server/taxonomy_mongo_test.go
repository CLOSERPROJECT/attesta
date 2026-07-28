package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoStoreEnsureTaxonomyIndexes(t *testing.T) {
	categories := &fakeMongoCollection{}
	subs := &fakeMongoCollection{}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
	}}
	store := &MongoStore{dbPort: db}

	if err := store.EnsureTaxonomyIndexes(t.Context()); err != nil {
		t.Fatalf("EnsureTaxonomyIndexes: %v", err)
	}
	if len(categories.createIndexesModels) != 1 || len(categories.createIndexesModels[0]) != 1 {
		t.Fatalf("categories indexes = %#v", categories.createIndexesModels)
	}
	catKeys := categories.createIndexesModels[0][0].Keys
	if !reflect.DeepEqual(catKeys, bson.D{{Key: "slug", Value: 1}}) {
		t.Fatalf("category index keys = %#v", catKeys)
	}
	if categories.createIndexesModels[0][0].Options == nil || !*categories.createIndexesModels[0][0].Options.Unique {
		t.Fatal("expected unique category slug index")
	}
	if len(subs.createIndexesModels) != 1 || len(subs.createIndexesModels[0]) != 1 {
		t.Fatalf("sub indexes = %#v", subs.createIndexesModels)
	}
	subKeys := subs.createIndexesModels[0][0].Keys
	wantSubKeys := bson.D{{Key: "categorySlug", Value: 1}, {Key: "slug", Value: 1}}
	if !reflect.DeepEqual(subKeys, wantSubKeys) {
		t.Fatalf("sub index keys = %#v", subKeys)
	}
}

func TestMongoStoreReplaceTaxonomyDeletesThenInserts(t *testing.T) {
	categories := &fakeMongoCollection{}
	subs := &fakeMongoCollection{}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
	}}
	store := &MongoStore{dbPort: db}

	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}
	if len(categories.deleteManyFilters) != 1 {
		t.Fatalf("expected categories DeleteMany, got %d", len(categories.deleteManyFilters))
	}
	if len(subs.deleteManyFilters) != 1 {
		t.Fatalf("expected sub_categories DeleteMany, got %d", len(subs.deleteManyFilters))
	}
	if len(categories.insertDocuments) != 1 {
		t.Fatalf("category inserts = %d", len(categories.insertDocuments))
	}
	if len(subs.insertDocuments) != 1 {
		t.Fatalf("sub inserts = %d", len(subs.insertDocuments))
	}
	cat := categories.insertDocuments[0].(Category)
	if cat.ID.IsZero() || cat.Slug != "supply-chain" {
		t.Fatalf("inserted category = %#v", cat)
	}
}

func TestMongoStoreDeleteCategoryRefusesChildren(t *testing.T) {
	categories := &fakeMongoCollection{}
	subs := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*SubCategory)) = SubCategory{Slug: "procurement", CategorySlug: "supply-chain"}
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
	}}
	store := &MongoStore{dbPort: db}

	if err := store.DeleteCategory(t.Context(), "supply-chain"); !errors.Is(err, ErrCategoryHasSubCategories) {
		t.Fatalf("err = %v, want ErrCategoryHasSubCategories", err)
	}
	if len(categories.deleteOneFilters) != 0 {
		t.Fatal("DeleteOne should not run when children exist")
	}
}

func TestMongoStoreDeleteCategoryWhenEmpty(t *testing.T) {
	categories := &fakeMongoCollection{
		deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		},
	}
	subs := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{err: mongo.ErrNoDocuments}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
	}}
	store := &MongoStore{dbPort: db}

	if err := store.DeleteCategory(t.Context(), "supply-chain"); err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}
	if len(categories.deleteOneFilters) != 1 {
		t.Fatalf("expected DeleteOne, got %d", len(categories.deleteOneFilters))
	}
}

func TestMongoStoreGetCategoryBySlug(t *testing.T) {
	want := Category{ID: primitive.NewObjectID(), Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability"}
	categories := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Category)) = want
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories: categories,
	}}
	store := &MongoStore{dbPort: db}

	got, err := store.GetCategoryBySlug(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("GetCategoryBySlug: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got %#v want %#v", *got, want)
	}
}

func TestMongoStoreListAndGetAndDeleteSubCategory(t *testing.T) {
	want := SubCategory{
		ID:           primitive.NewObjectID(),
		CategorySlug: "supply-chain",
		Slug:         "procurement",
		Name:         "Procurement",
		Icon:         "procurement-workflow",
		SortOrder:    1,
	}
	subs := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{want}}, nil
		},
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*SubCategory)) = want
				return nil
			}}
		},
		deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 1}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionSubCategories: subs,
	}}
	store := &MongoStore{dbPort: db}

	listed, err := store.ListSubCategories(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("ListSubCategories: %v", err)
	}
	if len(listed) != 1 || listed[0].Slug != "procurement" {
		t.Fatalf("listed = %#v", listed)
	}

	got, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement")
	if err != nil {
		t.Fatalf("GetSubCategoryBySlug: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got %#v", got)
	}

	if err := store.DeleteSubCategory(t.Context(), "supply-chain", "procurement"); err != nil {
		t.Fatalf("DeleteSubCategory: %v", err)
	}

	subs.deleteOneFn = func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
		return &mongo.DeleteResult{DeletedCount: 0}, nil
	}
	if err := store.DeleteSubCategory(t.Context(), "supply-chain", "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("missing delete err = %v", err)
	}
}

func TestMongoStoreListCategoriesUsesSort(t *testing.T) {
	want := []Category{
		{Slug: "recycling-and-recovery", SortOrder: 1},
		{Slug: "supply-chain", SortOrder: 2},
	}
	categories := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{want[0], want[1]}}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories: categories,
	}}
	store := &MongoStore{dbPort: db}

	got, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	if len(categories.findOptionsCalls) != 1 || len(categories.findOptionsCalls[0]) != 1 {
		t.Fatalf("find options = %#v", categories.findOptionsCalls)
	}
}

func TestMongoStoreTaxonomyErrorPaths(t *testing.T) {
	findErr := errors.New("find failed")
	indexErr := errors.New("index failed")
	deleteErr := errors.New("delete failed")
	insertErr := errors.New("insert failed")

	t.Run("EnsureTaxonomyIndexes category error", func(t *testing.T) {
		categories := &fakeMongoCollection{
			createIndexesFn: func(ctx context.Context, models []mongo.IndexModel) error { return indexErr },
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: &fakeMongoCollection{},
		}}}
		if err := store.EnsureTaxonomyIndexes(t.Context()); !errors.Is(err, indexErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("ListCategories find error", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, findErr
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.ListCategories(t.Context()); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("ListCategories decode skip", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{items: []interface{}{"bad"}}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		got, err := store.ListCategories(t.Context())
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got = %#v", got)
		}
	})

	t.Run("GetCategoryBySlug error", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.GetCategoryBySlug(t.Context(), "x"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("DeleteCategory child lookup error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    &fakeMongoCollection{},
			collectionSubCategories: subs,
		}}}
		if err := store.DeleteCategory(t.Context(), "x"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("DeleteCategory missing", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		categories := &fakeMongoCollection{
			deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return &mongo.DeleteResult{DeletedCount: 0}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: subs,
		}}}
		if err := store.DeleteCategory(t.Context(), "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("DeleteCategory delete error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		categories := &fakeMongoCollection{
			deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: subs,
		}}}
		if err := store.DeleteCategory(t.Context(), "x"); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("ListSubCategories find error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, findErr
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if _, err := store.ListSubCategories(t.Context(), "x"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("GetSubCategoryBySlug error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if _, err := store.GetSubCategoryBySlug(t.Context(), "x", "y"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("DeleteSubCategory error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			deleteOneFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if err := store.DeleteSubCategory(t.Context(), "x", "y"); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("ReplaceTaxonomy validation and write errors", func(t *testing.T) {
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    &fakeMongoCollection{},
			collectionSubCategories: &fakeMongoCollection{},
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), []Category{{Name: "x", Icon: "weee"}}, nil); err == nil {
			t.Fatal("expected missing slug")
		}
		if err := store.ReplaceTaxonomy(t.Context(), []Category{{Slug: "x", Icon: "bad"}}, nil); err == nil {
			t.Fatal("expected bad icon")
		}
		if err := store.ReplaceTaxonomy(t.Context(), nil, []SubCategory{{Slug: "y", Icon: "weee"}}); err == nil {
			t.Fatal("expected missing parent")
		}
		if err := store.ReplaceTaxonomy(t.Context(), nil, []SubCategory{{CategorySlug: "x", Slug: "y", Icon: "bad"}}); err == nil {
			t.Fatal("expected bad sub icon")
		}

		categories := &fakeMongoCollection{
			deleteManyFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: &fakeMongoCollection{},
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), nil, nil); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}

		categories = &fakeMongoCollection{}
		subs := &fakeMongoCollection{
			deleteManyFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: subs,
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), nil, nil); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}

		categories = &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: &fakeMongoCollection{},
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), []Category{{Slug: "x", Name: "X", Icon: "weee"}}, nil); !errors.Is(err, insertErr) {
			t.Fatalf("err = %v", err)
		}

		categories = &fakeMongoCollection{}
		subs = &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: subs,
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), nil, []SubCategory{{CategorySlug: "x", Slug: "y", Name: "Y", Icon: "weee"}}); !errors.Is(err, insertErr) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMemoryStoreReplaceTaxonomyValidation(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{{Name: "X", Icon: "weee"}}, nil); err == nil {
		t.Fatal("expected missing category slug")
	}
	if err := store.ReplaceTaxonomy(t.Context(), nil, []SubCategory{{Slug: "y", Icon: "weee"}}); err == nil {
		t.Fatal("expected missing parent slug")
	}
	if err := store.DeleteCategory(t.Context(), "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("err = %v", err)
	}

	// Cover ensureTaxonomyMaps nil-map branches via zero-value store.
	raw := &MemoryStore{}
	if err := raw.ReplaceTaxonomy(t.Context(), []Category{{Slug: "x", Name: "X", Icon: "weee", SortOrder: 1}}, nil); err != nil {
		t.Fatalf("ReplaceTaxonomy on zero store: %v", err)
	}
	cats, err := raw.ListCategories(t.Context())
	if err != nil || len(cats) != 1 {
		t.Fatalf("cats = %#v err=%v", cats, err)
	}
	all, err := raw.ListSubCategories(t.Context(), "")
	if err != nil || len(all) != 0 {
		t.Fatalf("subs = %#v err=%v", all, err)
	}
}

