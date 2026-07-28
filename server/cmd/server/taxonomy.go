package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionCategories    = "categories"
	collectionSubCategories = "sub_categories"
)

// ErrCategoryHasSubCategories is returned when DeleteCategory is refused because
// one or more Sub-categories still reference the Category slug.
var ErrCategoryHasSubCategories = errors.New("category has sub-categories")

// ErrInvalidTaxonomyIcon is returned when an icon key is not in the allowlist.
var ErrInvalidTaxonomyIcon = errors.New("invalid taxonomy icon")

// Category is a platform-global discovery bucket for browsing Streams.
type Category struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Icon      string             `bson:"icon"`
	Slug      string             `bson:"slug"`
	SortOrder int                `bson:"sortOrder"`
}

// SubCategory is a discovery leaf under exactly one Category.
type SubCategory struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name"`
	Icon         string             `bson:"icon"`
	Slug         string             `bson:"slug"`
	SortOrder    int                `bson:"sortOrder"`
	Description  string             `bson:"description,omitempty"`
	CategorySlug string             `bson:"categorySlug"`
}

// Taxonomy icon allowlist (24 keys from the Figma seed; unprefixed stems).
var taxonomyIconAllowlist = map[string]struct{}{
	"weee":                   {},
	"batch-traceability":     {},
	"quality-control":        {},
	"approval-workflow":      {},
	"photovoltaic-module":    {},
	"flat-screen":            {},
	"pcb":                    {},
	"led-lighting":           {},
	"power-electronics":      {},
	"shipment-tracking":      {},
	"supplier-onboarding":    {},
	"procurement-workflow":   {},
	"order-fulfillment":      {},
	"partner-onboarding":     {},
	"compliance-review":      {},
	"reporting-workflow":     {},
	"audit-workflow":         {},
	"certification-workflow": {},
	"inspection-workflow":    {},
	"data-verification":      {},
	"change-request":         {},
	"corrective-action":      {},
	"maintenance-workflow":   {},
	"issue-tracking":         {},
}

func validateTaxonomyIcon(icon string) error {
	key := strings.TrimSpace(icon)
	if _, ok := taxonomyIconAllowlist[key]; !ok {
		return fmt.Errorf("%w: %q", ErrInvalidTaxonomyIcon, icon)
	}
	return nil
}

func cloneCategory(category Category) Category {
	return category
}

func cloneSubCategory(sub SubCategory) SubCategory {
	return sub
}

func (s *MemoryStore) ensureTaxonomyMaps() {
	if s.categories == nil {
		s.categories = map[string]Category{}
	}
	if s.subCategories == nil {
		s.subCategories = map[string]SubCategory{}
	}
}

func subCategoryKey(categorySlug, slug string) string {
	return strings.TrimSpace(categorySlug) + "\x00" + strings.TrimSpace(slug)
}

func (s *MemoryStore) ListCategories(_ context.Context) ([]Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Category, 0, len(s.categories))
	for _, category := range s.categories {
		items = append(items, cloneCategory(category))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].Slug < items[j].Slug
	})
	return items, nil
}

func (s *MemoryStore) GetCategoryBySlug(_ context.Context, slug string) (*Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	category, ok := s.categories[strings.TrimSpace(slug)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneCategory(category)
	return &cloned, nil
}

func (s *MemoryStore) DeleteCategory(_ context.Context, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmed := strings.TrimSpace(slug)
	if _, ok := s.categories[trimmed]; !ok {
		return mongo.ErrNoDocuments
	}
	for _, sub := range s.subCategories {
		if sub.CategorySlug == trimmed {
			return ErrCategoryHasSubCategories
		}
	}
	delete(s.categories, trimmed)
	return nil
}

func (s *MemoryStore) ListSubCategories(_ context.Context, categorySlug string) ([]SubCategory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	parent := strings.TrimSpace(categorySlug)
	items := make([]SubCategory, 0)
	for _, sub := range s.subCategories {
		if parent != "" && sub.CategorySlug != parent {
			continue
		}
		items = append(items, cloneSubCategory(sub))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		if items[i].CategorySlug != items[j].CategorySlug {
			return items[i].CategorySlug < items[j].CategorySlug
		}
		return items[i].Slug < items[j].Slug
	})
	return items, nil
}

func (s *MemoryStore) GetSubCategoryBySlug(_ context.Context, categorySlug, slug string) (*SubCategory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subCategories[subCategoryKey(categorySlug, slug)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneSubCategory(sub)
	return &cloned, nil
}

func (s *MemoryStore) DeleteSubCategory(_ context.Context, categorySlug, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := subCategoryKey(categorySlug, slug)
	if _, ok := s.subCategories[key]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.subCategories, key)
	return nil
}

func (s *MemoryStore) EnsureTaxonomyIndexes(_ context.Context) error {
	return nil
}

func (s *MemoryStore) ReplaceTaxonomy(_ context.Context, categories []Category, subCategories []SubCategory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()

	nextCategories := make(map[string]Category, len(categories))
	for _, category := range categories {
		slug := strings.TrimSpace(category.Slug)
		if slug == "" {
			return fmt.Errorf("category slug is required")
		}
		if err := validateTaxonomyIcon(category.Icon); err != nil {
			return err
		}
		if category.ID.IsZero() {
			category.ID = primitive.NewObjectID()
		}
		category.Slug = slug
		category.Name = strings.TrimSpace(category.Name)
		category.Icon = strings.TrimSpace(category.Icon)
		nextCategories[slug] = cloneCategory(category)
	}

	nextSubs := make(map[string]SubCategory, len(subCategories))
	for _, sub := range subCategories {
		parent := strings.TrimSpace(sub.CategorySlug)
		slug := strings.TrimSpace(sub.Slug)
		if parent == "" || slug == "" {
			return fmt.Errorf("sub-category categorySlug and slug are required")
		}
		if err := validateTaxonomyIcon(sub.Icon); err != nil {
			return err
		}
		if sub.ID.IsZero() {
			sub.ID = primitive.NewObjectID()
		}
		sub.CategorySlug = parent
		sub.Slug = slug
		sub.Name = strings.TrimSpace(sub.Name)
		sub.Icon = strings.TrimSpace(sub.Icon)
		sub.Description = strings.TrimSpace(sub.Description)
		nextSubs[subCategoryKey(parent, slug)] = cloneSubCategory(sub)
	}

	s.categories = nextCategories
	s.subCategories = nextSubs
	return nil
}

func (s *MongoStore) ListCategories(ctx context.Context) ([]Category, error) {
	cursor, err := s.database().Collection(collectionCategories).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "slug", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []Category{}
	for cursor.Next(ctx) {
		var category Category
		if err := cursor.Decode(&category); err != nil {
			continue
		}
		items = append(items, category)
	}
	return items, nil
}

func (s *MongoStore) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	var category Category
	if err := s.database().Collection(collectionCategories).FindOne(
		ctx,
		bson.M{"slug": strings.TrimSpace(slug)},
	).Decode(&category); err != nil {
		return nil, err
	}
	return &category, nil
}

func (s *MongoStore) DeleteCategory(ctx context.Context, slug string) error {
	trimmed := strings.TrimSpace(slug)
	var child SubCategory
	err := s.database().Collection(collectionSubCategories).FindOne(
		ctx,
		bson.M{"categorySlug": trimmed},
	).Decode(&child)
	if err == nil {
		return ErrCategoryHasSubCategories
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	result, err := s.database().Collection(collectionCategories).DeleteOne(ctx, bson.M{"slug": trimmed})
	if err != nil {
		return err
	}
	if result != nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) ListSubCategories(ctx context.Context, categorySlug string) ([]SubCategory, error) {
	filter := bson.M{}
	if parent := strings.TrimSpace(categorySlug); parent != "" {
		filter["categorySlug"] = parent
	}
	cursor, err := s.database().Collection(collectionSubCategories).Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{
			{Key: "sortOrder", Value: 1},
			{Key: "categorySlug", Value: 1},
			{Key: "slug", Value: 1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []SubCategory{}
	for cursor.Next(ctx) {
		var sub SubCategory
		if err := cursor.Decode(&sub); err != nil {
			continue
		}
		items = append(items, sub)
	}
	return items, nil
}

func (s *MongoStore) GetSubCategoryBySlug(ctx context.Context, categorySlug, slug string) (*SubCategory, error) {
	var sub SubCategory
	if err := s.database().Collection(collectionSubCategories).FindOne(
		ctx,
		bson.M{
			"categorySlug": strings.TrimSpace(categorySlug),
			"slug":         strings.TrimSpace(slug),
		},
	).Decode(&sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *MongoStore) DeleteSubCategory(ctx context.Context, categorySlug, slug string) error {
	result, err := s.database().Collection(collectionSubCategories).DeleteOne(ctx, bson.M{
		"categorySlug": strings.TrimSpace(categorySlug),
		"slug":         strings.TrimSpace(slug),
	})
	if err != nil {
		return err
	}
	if result != nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) EnsureTaxonomyIndexes(ctx context.Context) error {
	if err := s.database().Collection(collectionCategories).CreateIndexes(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("categories_slug_unique"),
		},
	}); err != nil {
		return err
	}
	return s.database().Collection(collectionSubCategories).CreateIndexes(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "categorySlug", Value: 1}, {Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("sub_categories_parent_slug_unique"),
		},
	})
}

func (s *MongoStore) ReplaceTaxonomy(ctx context.Context, categories []Category, subCategories []SubCategory) error {
	preparedCategories := make([]Category, 0, len(categories))
	for _, category := range categories {
		slug := strings.TrimSpace(category.Slug)
		if slug == "" {
			return fmt.Errorf("category slug is required")
		}
		if err := validateTaxonomyIcon(category.Icon); err != nil {
			return err
		}
		if category.ID.IsZero() {
			category.ID = primitive.NewObjectID()
		}
		category.Slug = slug
		category.Name = strings.TrimSpace(category.Name)
		category.Icon = strings.TrimSpace(category.Icon)
		preparedCategories = append(preparedCategories, category)
	}

	preparedSubs := make([]SubCategory, 0, len(subCategories))
	for _, sub := range subCategories {
		parent := strings.TrimSpace(sub.CategorySlug)
		slug := strings.TrimSpace(sub.Slug)
		if parent == "" || slug == "" {
			return fmt.Errorf("sub-category categorySlug and slug are required")
		}
		if err := validateTaxonomyIcon(sub.Icon); err != nil {
			return err
		}
		if sub.ID.IsZero() {
			sub.ID = primitive.NewObjectID()
		}
		sub.CategorySlug = parent
		sub.Slug = slug
		sub.Name = strings.TrimSpace(sub.Name)
		sub.Icon = strings.TrimSpace(sub.Icon)
		sub.Description = strings.TrimSpace(sub.Description)
		preparedSubs = append(preparedSubs, sub)
	}

	if _, err := s.database().Collection(collectionCategories).DeleteMany(ctx, bson.M{}); err != nil {
		return err
	}
	if _, err := s.database().Collection(collectionSubCategories).DeleteMany(ctx, bson.M{}); err != nil {
		return err
	}
	for _, category := range preparedCategories {
		if _, err := s.database().Collection(collectionCategories).InsertOne(ctx, category); err != nil {
			return err
		}
	}
	for _, sub := range preparedSubs {
		if _, err := s.database().Collection(collectionSubCategories).InsertOne(ctx, sub); err != nil {
			return err
		}
	}
	return nil
}
