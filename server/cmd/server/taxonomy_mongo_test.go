package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func TestMongoStoreReplaceTaxonomyStagesThenRenames(t *testing.T) {
	liveCats := &fakeMongoCollection{}
	liveSubs := &fakeMongoCollection{}
	stagingCats := &fakeMongoCollection{}
	stagingSubs := &fakeMongoCollection{}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:           liveCats,
		collectionSubCategories:        liveSubs,
		collectionCategoriesStaging:    stagingCats,
		collectionSubCategoriesStaging: stagingSubs,
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
	if len(liveCats.deleteManyFilters) != 0 || len(liveCats.insertDocuments) != 0 {
		t.Fatal("live categories must not be written directly")
	}
	if len(liveSubs.deleteManyFilters) != 0 || len(liveSubs.insertDocuments) != 0 {
		t.Fatal("live sub_categories must not be written directly")
	}
	if len(stagingCats.deleteManyFilters) != 1 || len(stagingCats.insertDocuments) != 1 {
		t.Fatalf("staging categories delete=%d insert=%d", len(stagingCats.deleteManyFilters), len(stagingCats.insertDocuments))
	}
	if len(stagingSubs.deleteManyFilters) != 1 || len(stagingSubs.insertDocuments) != 1 {
		t.Fatalf("staging subs delete=%d insert=%d", len(stagingSubs.deleteManyFilters), len(stagingSubs.insertDocuments))
	}
	cat := stagingCats.insertDocuments[0].(Category)
	if cat.ID.IsZero() || cat.Slug != "supply-chain" {
		t.Fatalf("inserted category = %#v", cat)
	}
	if len(db.renameCalls) != 2 {
		t.Fatalf("renameCalls = %#v, want 2", db.renameCalls)
	}
	if db.renameCalls[0].from != collectionCategoriesStaging || db.renameCalls[0].to != collectionCategories || !db.renameCalls[0].dropTarget {
		t.Fatalf("categories rename = %#v", db.renameCalls[0])
	}
	if db.renameCalls[1].from != collectionSubCategoriesStaging || db.renameCalls[1].to != collectionSubCategories || !db.renameCalls[1].dropTarget {
		t.Fatalf("sub_categories rename = %#v", db.renameCalls[1])
	}
}

func TestMongoStoreTaxonomyRevision(t *testing.T) {
	t.Run("missing doc is zero", func(t *testing.T) {
		store := &MongoStore{dbPort: &fakeMongoDatabase{}}
		got, err := store.TaxonomyRevision(t.Context())
		if err != nil {
			t.Fatalf("TaxonomyRevision: %v", err)
		}
		if got != 0 {
			t.Fatalf("revision = %d, want 0", got)
		}
	})

	t.Run("returns stored n", func(t *testing.T) {
		meta := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{decodeFn: func(v interface{}) error {
					doc := v.(*struct {
						N int64 `bson:"n"`
					})
					doc.N = 7
					return nil
				}}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionTaxonomyMeta: meta,
		}}}
		got, err := store.TaxonomyRevision(t.Context())
		if err != nil {
			t.Fatalf("TaxonomyRevision: %v", err)
		}
		if got != 7 {
			t.Fatalf("revision = %d, want 7", got)
		}
	})

	t.Run("propagates find error", func(t *testing.T) {
		findErr := errors.New("meta find failed")
		meta := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionTaxonomyMeta: meta,
		}}}
		if _, err := store.TaxonomyRevision(t.Context()); !errors.Is(err, findErr) {
			t.Fatalf("err = %v, want %v", err, findErr)
		}
	})
}

func TestMongoStoreReplaceTaxonomyLeavesLiveIntactWhenStagingInsertFails(t *testing.T) {
	insertErr := errors.New("staging insert failed")
	liveCats := &fakeMongoCollection{}
	liveSubs := &fakeMongoCollection{}
	stagingCats := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return nil, insertErr
		},
	}
	stagingSubs := &fakeMongoCollection{}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:           liveCats,
		collectionSubCategories:        liveSubs,
		collectionCategoriesStaging:    stagingCats,
		collectionSubCategoriesStaging: stagingSubs,
	}}
	store := &MongoStore{dbPort: db}

	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, nil)
	if !errors.Is(err, insertErr) {
		t.Fatalf("err = %v, want %v", err, insertErr)
	}
	if len(liveCats.deleteManyFilters) != 0 || len(liveSubs.deleteManyFilters) != 0 {
		t.Fatalf("live collections must not be cleared on staging failure; cats=%d subs=%d",
			len(liveCats.deleteManyFilters), len(liveSubs.deleteManyFilters))
	}
	if len(liveCats.insertDocuments) != 0 || len(liveSubs.insertDocuments) != 0 {
		t.Fatal("live collections must not receive inserts on staging failure")
	}
	if len(db.renameCalls) != 0 {
		t.Fatalf("rename must not run on staging failure, got %#v", db.renameCalls)
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

	t.Run("ListCategories cursor error", func(t *testing.T) {
		cursorErr := errors.New("cursor iteration failed")
		categories := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{
					items: []interface{}{Category{Slug: "supply-chain", Name: "Supply Chain", Icon: "weee"}},
					err:   cursorErr,
				}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.ListCategories(t.Context()); !errors.Is(err, cursorErr) {
			t.Fatalf("err = %v, want %v", err, cursorErr)
		}
	})

	t.Run("ListCategories decode error", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{items: []interface{}{"bad"}}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.ListCategories(t.Context()); err == nil {
			t.Fatal("expected decode error")
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

	t.Run("ListSubCategories cursor error", func(t *testing.T) {
		cursorErr := errors.New("sub cursor iteration failed")
		subs := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{
					items: []interface{}{SubCategory{CategorySlug: "x", Slug: "y", Name: "Y", Icon: "weee"}},
					err:   cursorErr,
				}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if _, err := store.ListSubCategories(t.Context(), "x"); !errors.Is(err, cursorErr) {
			t.Fatalf("err = %v, want %v", err, cursorErr)
		}
	})

	t.Run("ListSubCategories decode error", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return &fakeAnyCursor{items: []interface{}{"bad"}}, nil
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if _, err := store.ListSubCategories(t.Context(), "x"); err == nil {
			t.Fatal("expected decode error")
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
			collectionCategoriesStaging:    &fakeMongoCollection{},
			collectionSubCategoriesStaging: &fakeMongoCollection{},
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

		stagingCats := &fakeMongoCollection{
			deleteManyFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategoriesStaging:    stagingCats,
			collectionSubCategoriesStaging: &fakeMongoCollection{},
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), nil, nil); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}

		stagingCats = &fakeMongoCollection{}
		stagingSubs := &fakeMongoCollection{
			deleteManyFn: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, deleteErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategoriesStaging:    stagingCats,
			collectionSubCategoriesStaging: stagingSubs,
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), nil, nil); !errors.Is(err, deleteErr) {
			t.Fatalf("err = %v", err)
		}

		stagingCats = &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategoriesStaging:    stagingCats,
			collectionSubCategoriesStaging: &fakeMongoCollection{},
		}}}
		if err := store.ReplaceTaxonomy(t.Context(), []Category{{Slug: "x", Name: "X", Icon: "weee"}}, nil); !errors.Is(err, insertErr) {
			t.Fatalf("err = %v", err)
		}

		stagingCats = &fakeMongoCollection{}
		stagingSubs = &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategoriesStaging:    stagingCats,
			collectionSubCategoriesStaging: stagingSubs,
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

func TestMongoStoreCreateUpdateReorderCategory(t *testing.T) {
	categories := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{err: mongo.ErrNoDocuments}
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories: categories,
	}}
	store := &MongoStore{dbPort: db}

	if _, err := store.CreateCategory(t.Context(), Category{Name: "X", Icon: "batch-traceability"}); err == nil {
		t.Fatal("expected missing slug")
	}
	if _, err := store.CreateCategory(t.Context(), Category{Slug: "x", Name: "X", Icon: "nope"}); err == nil {
		t.Fatal("expected invalid icon")
	}

	created, err := store.CreateCategory(t.Context(), Category{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if created.ID.IsZero() || created.SortOrder != 1 || created.Slug != "supply-chain" {
		t.Fatalf("created = %#v", created)
	}
	if len(categories.insertDocuments) != 1 {
		t.Fatalf("inserts = %d", len(categories.insertDocuments))
	}

	categories.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*Category)) = Category{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 3}
			return nil
		}}
	}
	created2, err := store.CreateCategory(t.Context(), Category{Slug: "recycling", Name: "Recycling", Icon: "batch-traceability"})
	if err != nil {
		t.Fatalf("CreateCategory second: %v", err)
	}
	if created2.SortOrder != 4 {
		t.Fatalf("sortOrder = %d, want 4", created2.SortOrder)
	}

	categories.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*Category)) = Category{Slug: "supply-chain", Name: "Updated", Icon: "batch-traceability", SortOrder: 1}
			return nil
		}}
	}
	updated, err := store.UpdateCategory(t.Context(), "supply-chain", "Updated", "batch-traceability")
	if err != nil {
		t.Fatalf("UpdateCategory: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("updated = %#v", updated)
	}
	if _, err := store.UpdateCategory(t.Context(), "supply-chain", "", "batch-traceability"); err == nil {
		t.Fatal("expected empty name")
	}

	categories.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
		return &fakeAnyCursor{items: []interface{}{
			Category{Slug: "a", SortOrder: 1},
			Category{Slug: "b", SortOrder: 2},
		}}, nil
	}
	if err := store.ReorderCategory(t.Context(), "a", "down"); err != nil {
		t.Fatalf("ReorderCategory: %v", err)
	}
	if len(categories.updateOneFilters) != 2 {
		t.Fatalf("updateOne calls = %d, want 2", len(categories.updateOneFilters))
	}
	if err := store.ReorderCategory(t.Context(), "missing", "down"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("missing reorder err = %v", err)
	}
}

func TestMongoStoreCreateUpdateReorderSubCategory(t *testing.T) {
	parent := Category{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability"}
	categories := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Category)) = parent
				return nil
			}}
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

	if _, err := store.CreateSubCategory(t.Context(), SubCategory{Slug: "x", Name: "X", Icon: "procurement-workflow"}); err == nil {
		t.Fatal("expected missing parent")
	}

	created, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow",
	})
	if err != nil {
		t.Fatalf("CreateSubCategory: %v", err)
	}
	if created.ID.IsZero() || created.SortOrder != 1 || created.CategorySlug != "supply-chain" {
		t.Fatalf("created = %#v", created)
	}

	subs.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*SubCategory)) = SubCategory{CategorySlug: "supply-chain", Slug: "procurement", SortOrder: 2}
			return nil
		}}
	}
	created2, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "shipping", Name: "Shipping", Icon: "procurement-workflow",
	})
	if err != nil {
		t.Fatalf("CreateSubCategory second: %v", err)
	}
	if created2.SortOrder != 3 {
		t.Fatalf("sortOrder = %d, want 3", created2.SortOrder)
	}

	subs.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*SubCategory)) = SubCategory{
				CategorySlug: "supply-chain", Slug: "procurement", Name: "Updated", Icon: "procurement-workflow", Description: "d",
			}
			return nil
		}}
	}
	updated, err := store.UpdateSubCategory(t.Context(), "supply-chain", "procurement", "Updated", "procurement-workflow", "d")
	if err != nil {
		t.Fatalf("UpdateSubCategory: %v", err)
	}
	if updated.Name != "Updated" || updated.Description != "d" {
		t.Fatalf("updated = %#v", updated)
	}
	if _, err := store.UpdateSubCategory(t.Context(), "supply-chain", "procurement", "", "procurement-workflow", ""); err == nil {
		t.Fatal("expected empty name")
	}

	subs.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
		return &fakeAnyCursor{items: []interface{}{
			SubCategory{CategorySlug: "supply-chain", Slug: "a", SortOrder: 1},
			SubCategory{CategorySlug: "supply-chain", Slug: "b", SortOrder: 2},
		}}, nil
	}
	if err := store.ReorderSubCategory(t.Context(), "supply-chain", "a", "down"); err != nil {
		t.Fatalf("ReorderSubCategory: %v", err)
	}
	if len(subs.updateOneFilters) != 2 {
		t.Fatalf("updateOne calls = %d, want 2", len(subs.updateOneFilters))
	}
	if err := store.ReorderSubCategory(t.Context(), "supply-chain", "missing", "up"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("missing reorder err = %v", err)
	}
}

func TestMongoStoreTaxonomyMutationErrorPaths(t *testing.T) {
	findErr := errors.New("find failed")
	insertErr := errors.New("insert failed")
	updateErr := errors.New("update failed")
	bumpColl := &fakeMongoCollection{
		updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return nil, errors.New("bump failed")
		},
	}

	t.Run("CreateCategory find max order error", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.CreateCategory(t.Context(), Category{Slug: "x", Name: "X", Icon: "batch-traceability"}); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("CreateCategory insert and bump errors", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, insertErr
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.CreateCategory(t.Context(), Category{Slug: "x", Name: "X", Icon: "batch-traceability"}); !errors.Is(err, insertErr) {
			t.Fatalf("err = %v", err)
		}

		categories.insertOneFn = nil
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:  categories,
			collectionTaxonomyMeta: bumpColl,
		}}}
		if _, err := store.CreateCategory(t.Context(), Category{Slug: "x", Name: "X", Icon: "batch-traceability"}); err == nil || !strings.Contains(err.Error(), "bump failed") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("UpdateCategory not found and errors", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findOneAndUpdateFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if _, err := store.UpdateCategory(t.Context(), "missing", "N", "batch-traceability"); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
		if _, err := store.UpdateCategory(t.Context(), "x", "N", "bad-icon"); err == nil {
			t.Fatal("expected invalid icon")
		}
		categories.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
			return fakeSingleResult{err: updateErr}
		}
		if _, err := store.UpdateCategory(t.Context(), "x", "N", "batch-traceability"); !errors.Is(err, updateErr) {
			t.Fatalf("err = %v", err)
		}
		categories.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Category)) = Category{Slug: "x", Name: "N", Icon: "batch-traceability"}
				return nil
			}}
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:   categories,
			collectionTaxonomyMeta: bumpColl,
		}}}
		if _, err := store.UpdateCategory(t.Context(), "x", "N", "batch-traceability"); err == nil || !strings.Contains(err.Error(), "bump failed") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("CreateSubCategory parent and insert errors", func(t *testing.T) {
		categories := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: &fakeMongoCollection{},
		}}}
		if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "missing", Slug: "s", Name: "S", Icon: "procurement-workflow"}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
		if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "bad"}); err == nil {
			t.Fatal("expected invalid icon")
		}

		categories.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Category)) = Category{Slug: "g"}
				return nil
			}}
		}
		subs := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: findErr}
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories:    categories,
			collectionSubCategories: subs,
		}}}
		if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "procurement-workflow"}); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}

		subs.findOneFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{err: mongo.ErrNoDocuments}
		}
		subs.insertOneFn = func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return nil, insertErr
		}
		if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "procurement-workflow"}); !errors.Is(err, insertErr) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("UpdateSubCategory and reorder errors", func(t *testing.T) {
		subs := &fakeMongoCollection{
			findOneAndUpdateFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
				return fakeSingleResult{err: mongo.ErrNoDocuments}
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "N", "procurement-workflow", ""); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("err = %v", err)
		}
		if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "N", "bad", ""); err == nil {
			t.Fatal("expected invalid icon")
		}
		subs.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
			return fakeSingleResult{err: updateErr}
		}
		if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "N", "procurement-workflow", ""); !errors.Is(err, updateErr) {
			t.Fatalf("err = %v", err)
		}

		categories := &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, findErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionCategories: categories,
		}}}
		if err := store.ReorderCategory(t.Context(), "a", "down"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
		categories.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{Category{Slug: "a", SortOrder: 1}}}, nil
		}
		if err := store.ReorderCategory(t.Context(), "a", "up"); !errors.Is(err, ErrTaxonomyReorderBoundary) {
			t.Fatalf("err = %v", err)
		}
		categories.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				Category{Slug: "a", SortOrder: 1},
				Category{Slug: "b", SortOrder: 2},
			}}, nil
		}
		categories.updateOneFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return nil, updateErr
		}
		if err := store.ReorderCategory(t.Context(), "a", "down"); !errors.Is(err, updateErr) {
			t.Fatalf("err = %v", err)
		}

		subs = &fakeMongoCollection{
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, findErr
			},
		}
		store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionSubCategories: subs,
		}}}
		if err := store.ReorderSubCategory(t.Context(), "g", "a", "down"); !errors.Is(err, findErr) {
			t.Fatalf("err = %v", err)
		}
		subs.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{SubCategory{Slug: "a", SortOrder: 1}}}, nil
		}
		if err := store.ReorderSubCategory(t.Context(), "g", "a", "down"); !errors.Is(err, ErrTaxonomyReorderBoundary) {
			t.Fatalf("err = %v", err)
		}
		subs.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				SubCategory{Slug: "a", SortOrder: 1},
				SubCategory{Slug: "b", SortOrder: 2},
			}}, nil
		}
		subs.updateOneFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return nil, updateErr
		}
		if err := store.ReorderSubCategory(t.Context(), "g", "a", "down"); !errors.Is(err, updateErr) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMongoStoreCreateDuplicateAndSecondReorderUpdate(t *testing.T) {
	dup := mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "E11000"}}}
	categories := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{err: mongo.ErrNoDocuments}
		},
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return nil, dup
		},
	}
	store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories: categories,
	}}}
	if _, err := store.CreateCategory(t.Context(), Category{Slug: "x", Name: "X", Icon: "batch-traceability"}); !errors.Is(err, ErrTaxonomySlugExists) {
		t.Fatalf("err = %v", err)
	}

	categories.findFn = func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
		return &fakeAnyCursor{items: []interface{}{
			Category{Slug: "a", SortOrder: 1},
			Category{Slug: "b", SortOrder: 2},
		}}, nil
	}
	calls := 0
	categories.updateOneFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
		calls++
		if calls == 1 {
			return &mongo.UpdateResult{}, nil
		}
		return nil, errors.New("second update failed")
	}
	categories.insertOneFn = nil
	if err := store.ReorderCategory(t.Context(), "a", "down"); err == nil || !strings.Contains(err.Error(), "second update failed") {
		t.Fatalf("err = %v", err)
	}

	subs := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				SubCategory{Slug: "a", SortOrder: 1},
				SubCategory{Slug: "b", SortOrder: 2},
			}}, nil
		},
	}
	calls = 0
	subs.updateOneFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
		calls++
		if calls == 1 {
			return &mongo.UpdateResult{}, nil
		}
		return nil, errors.New("second sub update failed")
	}
	store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionSubCategories: subs,
	}}}
	if err := store.ReorderSubCategory(t.Context(), "g", "a", "down"); err == nil || !strings.Contains(err.Error(), "second sub update failed") {
		t.Fatalf("err = %v", err)
	}

	// CreateSubCategory duplicate + bump failure after insert
	categories = &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Category)) = Category{Slug: "g"}
				return nil
			}}
		},
	}
	subs = &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{err: mongo.ErrNoDocuments}
		},
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return nil, dup
		},
	}
	store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
	}}}
	if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "procurement-workflow"}); !errors.Is(err, ErrTaxonomySlugExists) {
		t.Fatalf("err = %v", err)
	}

	subs.insertOneFn = nil
	bumpColl := &fakeMongoCollection{
		updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			return nil, errors.New("bump failed")
		},
	}
	store = &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionCategories:    categories,
		collectionSubCategories: subs,
		collectionTaxonomyMeta:  bumpColl,
	}}}
	if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "procurement-workflow"}); err == nil || !strings.Contains(err.Error(), "bump failed") {
		t.Fatalf("err = %v", err)
	}

	subs.findOneAndUpdateFn = func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
		return fakeSingleResult{decodeFn: func(v interface{}) error {
			*(v.(*SubCategory)) = SubCategory{Slug: "s", Name: "S"}
			return nil
		}}
	}
	if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "S", "procurement-workflow", ""); err == nil || !strings.Contains(err.Error(), "bump failed") {
		t.Fatalf("err = %v", err)
	}
}
