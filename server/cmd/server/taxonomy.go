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
	collectionCategories           = "categories"
	collectionSubCategories        = "sub_categories"
	collectionCategoriesStaging    = "categories__next"
	collectionSubCategoriesStaging = "sub_categories__next"
	collectionTaxonomyMeta         = "taxonomy_meta"
	taxonomyRevisionDocID          = "revision"
)

// ErrCategoryHasSubCategories is returned when DeleteCategory is refused because
// one or more Sub-categories still reference the Category slug.
var ErrCategoryHasSubCategories = errors.New("category has sub-categories")

// ErrTaxonomySlugExists is returned when a category or sub-category slug collides.
var ErrTaxonomySlugExists = errors.New("taxonomy slug already exists")

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

func (s *MemoryStore) CreateCategory(_ context.Context, category Category) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	slug := strings.TrimSpace(category.Slug)
	name := strings.TrimSpace(category.Name)
	icon := strings.TrimSpace(category.Icon)
	if slug == "" || name == "" {
		return Category{}, fmt.Errorf("category slug and name are required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}
	if _, exists := s.categories[slug]; exists {
		return Category{}, ErrTaxonomySlugExists
	}
	maxOrder := 0
	for _, c := range s.categories {
		if c.SortOrder > maxOrder {
			maxOrder = c.SortOrder
		}
	}
	if category.ID.IsZero() {
		category.ID = primitive.NewObjectID()
	}
	category.Slug, category.Name, category.Icon = slug, name, icon
	category.SortOrder = maxOrder + 1
	s.categories[slug] = cloneCategory(category)
	s.taxonomyRevision++
	return cloneCategory(category), nil
}

func (s *MemoryStore) UpdateCategory(_ context.Context, slug, name, icon string) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	trimmed := strings.TrimSpace(slug)
	existing, ok := s.categories[trimmed]
	if !ok {
		return Category{}, mongo.ErrNoDocuments
	}
	name = strings.TrimSpace(name)
	icon = strings.TrimSpace(icon)
	if name == "" {
		return Category{}, fmt.Errorf("category name is required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}
	existing.Name, existing.Icon = name, icon
	s.categories[trimmed] = cloneCategory(existing)
	s.taxonomyRevision++
	return cloneCategory(existing), nil
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
	s.taxonomyRevision++
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

func (s *MemoryStore) CreateSubCategory(_ context.Context, sub SubCategory) (SubCategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	parent := strings.TrimSpace(sub.CategorySlug)
	slug := strings.TrimSpace(sub.Slug)
	name := strings.TrimSpace(sub.Name)
	icon := strings.TrimSpace(sub.Icon)
	description := strings.TrimSpace(sub.Description)
	if parent == "" || slug == "" || name == "" {
		return SubCategory{}, fmt.Errorf("sub-category categorySlug, slug, and name are required")
	}
	if _, ok := s.categories[parent]; !ok {
		return SubCategory{}, mongo.ErrNoDocuments
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return SubCategory{}, err
	}
	key := subCategoryKey(parent, slug)
	if _, exists := s.subCategories[key]; exists {
		return SubCategory{}, ErrTaxonomySlugExists
	}
	maxOrder := 0
	for _, existing := range s.subCategories {
		if existing.CategorySlug == parent && existing.SortOrder > maxOrder {
			maxOrder = existing.SortOrder
		}
	}
	if sub.ID.IsZero() {
		sub.ID = primitive.NewObjectID()
	}
	sub.CategorySlug, sub.Slug, sub.Name, sub.Icon, sub.Description = parent, slug, name, icon, description
	sub.SortOrder = maxOrder + 1
	s.subCategories[key] = cloneSubCategory(sub)
	s.taxonomyRevision++
	return cloneSubCategory(sub), nil
}

func (s *MemoryStore) UpdateSubCategory(_ context.Context, categorySlug, slug, name, icon, description string) (SubCategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	key := subCategoryKey(categorySlug, slug)
	existing, ok := s.subCategories[key]
	if !ok {
		return SubCategory{}, mongo.ErrNoDocuments
	}
	name = strings.TrimSpace(name)
	icon = strings.TrimSpace(icon)
	description = strings.TrimSpace(description)
	if name == "" {
		return SubCategory{}, fmt.Errorf("sub-category name is required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return SubCategory{}, err
	}
	existing.Name, existing.Icon, existing.Description = name, icon, description
	s.subCategories[key] = cloneSubCategory(existing)
	s.taxonomyRevision++
	return cloneSubCategory(existing), nil
}

func (s *MemoryStore) DeleteSubCategory(_ context.Context, categorySlug, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := subCategoryKey(categorySlug, slug)
	if _, ok := s.subCategories[key]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.subCategories, key)
	s.taxonomyRevision++
	return nil
}

func (s *MemoryStore) EnsureTaxonomyIndexes(_ context.Context) error {
	return nil
}

func (s *MemoryStore) TaxonomyRevision(_ context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taxonomyRevision, nil
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
	s.taxonomyRevision++
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
			return nil, err
		}
		items = append(items, category)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
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

func (s *MongoStore) CreateCategory(ctx context.Context, category Category) (Category, error) {
	slug := strings.TrimSpace(category.Slug)
	name := strings.TrimSpace(category.Name)
	icon := strings.TrimSpace(category.Icon)
	if slug == "" || name == "" {
		return Category{}, fmt.Errorf("category slug and name are required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}

	maxOrder := 0
	var top Category
	err := s.database().Collection(collectionCategories).FindOne(
		ctx,
		bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: -1}}),
	).Decode(&top)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return Category{}, err
	}
	if err == nil {
		maxOrder = top.SortOrder
	}

	if category.ID.IsZero() {
		category.ID = primitive.NewObjectID()
	}
	category.Slug, category.Name, category.Icon = slug, name, icon
	category.SortOrder = maxOrder + 1

	if _, err := s.database().Collection(collectionCategories).InsertOne(ctx, category); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return Category{}, ErrTaxonomySlugExists
		}
		return Category{}, err
	}
	if err := s.bumpTaxonomyRevision(ctx); err != nil {
		return Category{}, err
	}
	return category, nil
}

func (s *MongoStore) UpdateCategory(ctx context.Context, slug, name, icon string) (Category, error) {
	trimmed := strings.TrimSpace(slug)
	name = strings.TrimSpace(name)
	icon = strings.TrimSpace(icon)
	if name == "" {
		return Category{}, fmt.Errorf("category name is required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}

	var updated Category
	err := s.database().Collection(collectionCategories).FindOneAndUpdate(
		ctx,
		bson.M{"slug": trimmed},
		bson.M{"$set": bson.M{"name": name, "icon": icon}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Category{}, mongo.ErrNoDocuments
	}
	if err != nil {
		return Category{}, err
	}
	if err := s.bumpTaxonomyRevision(ctx); err != nil {
		return Category{}, err
	}
	return updated, nil
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
	return s.bumpTaxonomyRevision(ctx)
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
			return nil, err
		}
		items = append(items, sub)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
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

func (s *MongoStore) CreateSubCategory(ctx context.Context, sub SubCategory) (SubCategory, error) {
	parent := strings.TrimSpace(sub.CategorySlug)
	slug := strings.TrimSpace(sub.Slug)
	name := strings.TrimSpace(sub.Name)
	icon := strings.TrimSpace(sub.Icon)
	description := strings.TrimSpace(sub.Description)
	if parent == "" || slug == "" || name == "" {
		return SubCategory{}, fmt.Errorf("sub-category categorySlug, slug, and name are required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return SubCategory{}, err
	}

	if _, err := s.GetCategoryBySlug(ctx, parent); err != nil {
		return SubCategory{}, err
	}

	maxOrder := 0
	var top SubCategory
	err := s.database().Collection(collectionSubCategories).FindOne(
		ctx,
		bson.M{"categorySlug": parent},
		options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: -1}}),
	).Decode(&top)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return SubCategory{}, err
	}
	if err == nil {
		maxOrder = top.SortOrder
	}

	if sub.ID.IsZero() {
		sub.ID = primitive.NewObjectID()
	}
	sub.CategorySlug, sub.Slug, sub.Name, sub.Icon, sub.Description = parent, slug, name, icon, description
	sub.SortOrder = maxOrder + 1

	if _, err := s.database().Collection(collectionSubCategories).InsertOne(ctx, sub); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return SubCategory{}, ErrTaxonomySlugExists
		}
		return SubCategory{}, err
	}
	if err := s.bumpTaxonomyRevision(ctx); err != nil {
		return SubCategory{}, err
	}
	return sub, nil
}

func (s *MongoStore) UpdateSubCategory(ctx context.Context, categorySlug, slug, name, icon, description string) (SubCategory, error) {
	parent := strings.TrimSpace(categorySlug)
	trimmed := strings.TrimSpace(slug)
	name = strings.TrimSpace(name)
	icon = strings.TrimSpace(icon)
	description = strings.TrimSpace(description)
	if name == "" {
		return SubCategory{}, fmt.Errorf("sub-category name is required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return SubCategory{}, err
	}

	var updated SubCategory
	err := s.database().Collection(collectionSubCategories).FindOneAndUpdate(
		ctx,
		bson.M{"categorySlug": parent, "slug": trimmed},
		bson.M{"$set": bson.M{"name": name, "icon": icon, "description": description}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return SubCategory{}, mongo.ErrNoDocuments
	}
	if err != nil {
		return SubCategory{}, err
	}
	if err := s.bumpTaxonomyRevision(ctx); err != nil {
		return SubCategory{}, err
	}
	return updated, nil
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
	return s.bumpTaxonomyRevision(ctx)
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

	db := s.database()
	stagingCats := db.Collection(collectionCategoriesStaging)
	stagingSubs := db.Collection(collectionSubCategoriesStaging)

	if _, err := stagingCats.DeleteMany(ctx, bson.M{}); err != nil {
		return err
	}
	if _, err := stagingSubs.DeleteMany(ctx, bson.M{}); err != nil {
		return err
	}
	for _, category := range preparedCategories {
		if _, err := stagingCats.InsertOne(ctx, category); err != nil {
			return err
		}
	}
	for _, sub := range preparedSubs {
		if _, err := stagingSubs.InsertOne(ctx, sub); err != nil {
			return err
		}
	}
	if err := stagingCats.CreateIndexes(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("categories_slug_unique"),
		},
	}); err != nil {
		return err
	}
	if err := stagingSubs.CreateIndexes(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "categorySlug", Value: 1}, {Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("sub_categories_parent_slug_unique"),
		},
	}); err != nil {
		return err
	}

	// Promote staging only after both collections are fully written.
	if err := db.RenameCollection(ctx, collectionCategoriesStaging, collectionCategories, true); err != nil {
		return err
	}
	if err := db.RenameCollection(ctx, collectionSubCategoriesStaging, collectionSubCategories, true); err != nil {
		return err
	}
	return s.bumpTaxonomyRevision(ctx)
}

func (s *MongoStore) TaxonomyRevision(ctx context.Context) (int64, error) {
	var doc struct {
		N int64 `bson:"n"`
	}
	err := s.database().Collection(collectionTaxonomyMeta).FindOne(
		ctx,
		bson.M{"_id": taxonomyRevisionDocID},
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return doc.N, nil
}

func (s *MongoStore) bumpTaxonomyRevision(ctx context.Context) error {
	_, err := s.database().Collection(collectionTaxonomyMeta).UpdateOne(
		ctx,
		bson.M{"_id": taxonomyRevisionDocID},
		bson.M{"$inc": bson.M{"n": int64(1)}},
		options.Update().SetUpsert(true),
	)
	return err
}
