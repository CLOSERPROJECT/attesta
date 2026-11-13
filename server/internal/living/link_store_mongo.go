package living

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionLivingSubmissionLinks = "living_submission_links"

var _ LinkStore = (*MongoLinkStore)(nil)

// MongoLinkStore persists post-Commit links in a collection separate from Domain facts.
type MongoLinkStore struct {
	db *mongo.Database
}

func NewMongoLinkStore(db *mongo.Database) *MongoLinkStore {
	return &MongoLinkStore{db: db}
}

type mongoSubmissionLinksDoc struct {
	ID              string         `bson:"_id"`
	Org             string         `bson:"org"`
	JobID           string         `bson:"jobId"`
	JobInstanceID   string         `bson:"jobInstanceId"`
	TaskID          string         `bson:"taskId"`
	GroupID         string         `bson:"groupId,omitempty"`
	GroupInstanceID string         `bson:"groupInstanceId,omitempty"`
	Timestamp       string         `bson:"timestamp"`
	Links           []HttpLinkItem `bson:"links"`
}

func (s *MongoLinkStore) coll() *mongo.Collection {
	return s.db.Collection(collectionLivingSubmissionLinks)
}

func (s *MongoLinkStore) Put(ctx context.Context, org string, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error {
	key, err := linkStoreKey(org, addr, timestamp)
	if err != nil {
		return err
	}
	addr = addr.normalized()
	org = strings.TrimSpace(org)
	timestamp = strings.TrimSpace(timestamp)
	doc := mongoSubmissionLinksDoc{
		ID:              key,
		Org:             org,
		JobID:           addr.JobID,
		JobInstanceID:   addr.JobInstanceID,
		TaskID:          addr.TaskID,
		GroupID:         addr.GroupID,
		GroupInstanceID: addr.GroupInstanceID,
		Timestamp:       timestamp,
		Links:           cloneHTTPLinks(links),
	}
	if doc.Links == nil {
		doc.Links = []HttpLinkItem{}
	}
	_, err = s.coll().ReplaceOne(ctx, bson.M{"_id": key}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("living mongo links put: %w", err)
	}
	return nil
}

func (s *MongoLinkStore) Get(ctx context.Context, org string, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error) {
	key, err := linkStoreKey(org, addr, timestamp)
	if err != nil {
		return nil, err
	}
	var doc mongoSubmissionLinksDoc
	err = s.coll().FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("living mongo links get: %w", err)
	}
	return cloneHTTPLinks(doc.Links), nil
}
