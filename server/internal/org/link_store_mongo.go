package org

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionOrgSubmissionLinks = "org_submission_links"

var _ LinkStore = (*MongoLinkStore)(nil)

// MongoLinkStore persists post-Commit links in a collection separate from Domain facts.
type MongoLinkStore struct {
	db *mongo.Database
}

func NewMongoLinkStore(db *mongo.Database) *MongoLinkStore {
	return &MongoLinkStore{db: db}
}

type mongoSubmissionLinksDoc struct {
	ID            string         `bson:"_id"`
	Org           string         `bson:"org"`
	Stream        string         `bson:"stream"`
	JobID         string         `bson:"jobId"`
	JobInstanceID string         `bson:"jobInstanceId"`
	TaskID        string         `bson:"taskId"`
	Timestamp     string         `bson:"timestamp"`
	Links         []HttpLinkItem `bson:"links"`
}

func (s *MongoLinkStore) coll() *mongo.Collection {
	return s.db.Collection(collectionOrgSubmissionLinks)
}

func (s *MongoLinkStore) Put(ctx context.Context, scope Scope, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error {
	key, err := linkStoreKey(scope, addr, timestamp)
	if err != nil {
		return err
	}
	scope, err = scope.Normalize()
	if err != nil {
		return err
	}
	addr = addr.normalized()
	timestamp = strings.TrimSpace(timestamp)
	doc := mongoSubmissionLinksDoc{
		ID:            key,
		Org:           scope.Org,
		Stream:        scope.Stream,
		JobID:         addr.JobID,
		JobInstanceID: addr.JobInstanceID,
		TaskID:        addr.TaskID,
		Timestamp:     timestamp,
		Links:         cloneHTTPLinks(links),
	}
	if doc.Links == nil {
		doc.Links = []HttpLinkItem{}
	}
	_, err = s.coll().ReplaceOne(ctx, bson.M{"_id": key}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("org mongo links put: %w", err)
	}
	return nil
}

func (s *MongoLinkStore) Get(ctx context.Context, scope Scope, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error) {
	key, err := linkStoreKey(scope, addr, timestamp)
	if err != nil {
		return nil, err
	}
	var doc mongoSubmissionLinksDoc
	err = s.coll().FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("org mongo links get: %w", err)
	}
	return cloneHTTPLinks(doc.Links), nil
}
