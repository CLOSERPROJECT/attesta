package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionOrgDomains      = "org_domains"
	collectionOrgConfigs      = "org_configs"
	collectionOrgActiveConfig = "org_active_config"
)

const maxDomainUpdateAttempts = 5

var (
	_ DomainStore = (*MongoDomainStore)(nil)
	_ ConfigStore = (*MongoConfigStore)(nil)
)

// MongoDomainStore persists Domain documents in MongoDB (one per org+stream).
type MongoDomainStore struct {
	db *mongo.Database
}

func NewMongoDomainStore(db *mongo.Database) *MongoDomainStore {
	return &MongoDomainStore{db: db}
}

type mongoDomainDoc struct {
	ID           string                               `bson:"_id"`
	Org          string                               `bson:"org"`
	Stream       string                               `bson:"stream"`
	Entities     map[string]map[string]map[string]any `bson:"entities"`
	Submissions  []mongoSubmissionDoc                 `bson:"submissions"`
	JobInstances []JobInstance                        `bson:"jobInstances"`
	// Bumped on every write; Update compares it to detect concurrent writers.
	Revision int64 `bson:"revision"`
}

type mongoSubmissionDoc struct {
	JobID         string         `bson:"jobId"`
	TaskID        string         `bson:"taskId"`
	JobInstanceID string         `bson:"jobInstanceId,omitempty"`
	Timestamp     string         `bson:"timestamp"`
	UserID        string         `bson:"userId"`
	UserName      string         `bson:"userName,omitempty"`
	Facts         map[string]any `bson:"facts"`
	IDsTouched    []string       `bson:"idsTouched"`
}

func (s *MongoDomainStore) coll() *mongo.Collection {
	return s.db.Collection(collectionOrgDomains)
}

func (s *MongoDomainStore) Get(ctx context.Context, scope Scope) (Domain, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return EmptyDomain(), ErrNotFound
	}
	doc, err := s.find(ctx, scope)
	if err != nil {
		return EmptyDomain(), err
	}
	return domainFromMongoDoc(doc), nil
}

func (s *MongoDomainStore) find(ctx context.Context, scope Scope) (mongoDomainDoc, error) {
	var doc mongoDomainDoc
	err := s.coll().FindOne(ctx, bson.M{"_id": scopeKey(scope)}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongoDomainDoc{}, ErrNotFound
	}
	if err != nil {
		return mongoDomainDoc{}, fmt.Errorf("org mongo domain get: %w", err)
	}
	return doc, nil
}

func (s *MongoDomainStore) Put(ctx context.Context, scope Scope, domain Domain) error {
	scope, err := scope.Normalize()
	if err != nil {
		return fmt.Errorf("org mongo domain put: %w", err)
	}
	_, err = s.coll().UpdateOne(ctx, bson.M{"_id": scopeKey(scope)}, domainWriteUpdate(scope, domain), options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("org mongo domain put: %w", err)
	}
	return nil
}

func (s *MongoDomainStore) Update(ctx context.Context, scope Scope, fn func(Domain) (Domain, error)) (Domain, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return Domain{}, fmt.Errorf("org mongo domain update: %w", err)
	}
	for range maxDomainUpdateAttempts {
		doc, err := s.find(ctx, scope)
		missing := errors.Is(err, ErrNotFound)
		if err != nil && !missing {
			return Domain{}, err
		}
		current := EmptyDomain()
		if !missing {
			current = domainFromMongoDoc(doc)
		}
		next, err := fn(current)
		if err != nil {
			return Domain{}, err
		}
		// Upsert only when the doc was missing: a concurrent insert then trips the _id unique index.
		res, err := s.coll().UpdateOne(ctx, domainRevisionFilter(scope, doc.Revision), domainWriteUpdate(scope, next), options.Update().SetUpsert(missing))
		if mongo.IsDuplicateKeyError(err) {
			continue
		}
		if err != nil {
			return Domain{}, fmt.Errorf("org mongo domain update: %w", err)
		}
		if res.MatchedCount == 0 && res.UpsertedCount == 0 {
			continue
		}
		return next, nil
	}
	return Domain{}, ErrDomainConflict
}

// domainRevisionFilter treats a missing revision field (docs written before
// revisions existed) as revision 0.
func domainRevisionFilter(scope Scope, revision int64) bson.M {
	id := scopeKey(scope)
	if revision == 0 {
		return bson.M{"_id": id, "revision": bson.M{"$in": bson.A{int64(0), nil}}}
	}
	return bson.M{"_id": id, "revision": revision}
}

func domainWriteUpdate(scope Scope, domain Domain) bson.M {
	doc := domainToMongoDoc(scope, domain)
	return bson.M{
		"$set": bson.M{
			"org":          doc.Org,
			"stream":       doc.Stream,
			"entities":     doc.Entities,
			"submissions":  doc.Submissions,
			"jobInstances": doc.JobInstances,
		},
		"$inc": bson.M{"revision": int64(1)},
	}
}

// MongoConfigStore persists versioned Org Config documents and Active Config pointers.
type MongoConfigStore struct {
	db *mongo.Database
}

func NewMongoConfigStore(db *mongo.Database) *MongoConfigStore {
	return &MongoConfigStore{db: db}
}

type mongoOrgConfigDoc struct {
	ID      string `bson:"_id"`
	Org     string `bson:"org"`
	Stream  string `bson:"stream"`
	Version string `bson:"version"`
	Payload []byte `bson:"payload"`
}

type mongoActiveConfigDoc struct {
	ID      string `bson:"_id"`
	Org     string `bson:"org"`
	Stream  string `bson:"stream"`
	Version string `bson:"version"`
}

func (s *MongoConfigStore) configs() *mongo.Collection {
	return s.db.Collection(collectionOrgConfigs)
}

func (s *MongoConfigStore) active() *mongo.Collection {
	return s.db.Collection(collectionOrgActiveConfig)
}

func (s *MongoConfigStore) Get(ctx context.Context, scope Scope, version string) (OrgConfig, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return OrgConfig{}, ErrNotFound
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return OrgConfig{}, ErrNotFound
	}
	var doc mongoOrgConfigDoc
	err = s.configs().FindOne(ctx, bson.M{"_id": configKey(scope, version)}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return OrgConfig{}, ErrNotFound
	}
	if err != nil {
		return OrgConfig{}, fmt.Errorf("org mongo config get: %w", err)
	}
	var cfg OrgConfig
	if err := json.Unmarshal(doc.Payload, &cfg); err != nil {
		return OrgConfig{}, fmt.Errorf("org mongo config decode: %w", err)
	}
	normalizeOrgConfigJobs(&cfg)
	cfg.Org = scope.Org
	cfg.Version = version
	if cfg.Stream == nil {
		cfg.Stream = &StreamConfig{Key: scope.Stream}
	} else {
		cfg.Stream.Key = scope.Stream
	}
	return cfg, nil
}

func (s *MongoConfigStore) Put(ctx context.Context, scope Scope, version string, cfg OrgConfig) error {
	scope, err := scope.Normalize()
	if err != nil {
		return fmt.Errorf("org mongo config put: %w", err)
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("org mongo config put: org, stream and version required")
	}
	cfg.Org = scope.Org
	cfg.Version = version
	if cfg.Stream == nil {
		cfg.Stream = &StreamConfig{Key: scope.Stream}
	} else {
		cfg.Stream.Key = scope.Stream
	}
	payload, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("org mongo config encode: %w", err)
	}
	doc := mongoOrgConfigDoc{
		ID:      configKey(scope, version),
		Org:     scope.Org,
		Stream:  scope.Stream,
		Version: version,
		Payload: payload,
	}
	_, err = s.configs().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("org mongo config put: %w", err)
	}
	return nil
}

func (s *MongoConfigStore) ActiveVersion(ctx context.Context, scope Scope) (string, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return "", ErrNotFound
	}
	var doc mongoActiveConfigDoc
	err = s.active().FindOne(ctx, bson.M{"_id": scopeKey(scope)}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("org mongo active get: %w", err)
	}
	if strings.TrimSpace(doc.Version) == "" {
		return "", ErrNotFound
	}
	return doc.Version, nil
}

func (s *MongoConfigStore) SetActiveVersion(ctx context.Context, scope Scope, version string) error {
	scope, err := scope.Normalize()
	if err != nil {
		return fmt.Errorf("org mongo active set: %w", err)
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("org mongo active set: org, stream and version required")
	}
	err = s.configs().FindOne(ctx, bson.M{"_id": configKey(scope, version)}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("org mongo active set: lookup: %w", err)
	}
	doc := mongoActiveConfigDoc{
		ID:      scopeKey(scope),
		Org:     scope.Org,
		Stream:  scope.Stream,
		Version: version,
	}
	_, err = s.active().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("org mongo active set: %w", err)
	}
	return nil
}

func (s *MongoConfigStore) ListStreams(ctx context.Context, org string) ([]string, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return nil, fmt.Errorf("org: org required")
	}
	cur, err := s.active().Find(ctx, bson.M{"org": org})
	if err != nil {
		return nil, fmt.Errorf("org mongo list streams: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]string, 0)
	for cur.Next(ctx) {
		var doc mongoActiveConfigDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("org mongo list streams: decode: %w", err)
		}
		stream := strings.TrimSpace(doc.Stream)
		if stream == "" || strings.TrimSpace(doc.Version) == "" {
			continue
		}
		out = append(out, stream)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("org mongo list streams: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

func domainToMongoDoc(scope Scope, domain Domain) mongoDomainDoc {
	ensureEntities(&domain)
	doc := mongoDomainDoc{
		ID:           scopeKey(scope),
		Org:          scope.Org,
		Stream:       scope.Stream,
		Entities:     make(map[string]map[string]map[string]any, len(domain.Entities)),
		Submissions:  encodeSubmissions(domain.Submissions),
		JobInstances: append([]JobInstance(nil), domain.JobInstances...),
	}
	for kind, m := range domain.Entities {
		doc.Entities[encodeMongoKey(kind)] = encodeEntityMap(m)
	}
	if doc.Entities == nil {
		doc.Entities = map[string]map[string]map[string]any{}
	}
	if doc.Submissions == nil {
		doc.Submissions = []mongoSubmissionDoc{}
	}
	if doc.JobInstances == nil {
		doc.JobInstances = []JobInstance{}
	}
	return doc
}

func domainFromMongoDoc(doc mongoDomainDoc) Domain {
	out := EmptyDomain()
	out.Submissions = decodeSubmissions(doc.Submissions)
	out.JobInstances = append([]JobInstance(nil), doc.JobInstances...)
	for kind, m := range doc.Entities {
		out.Entities[decodeMongoKey(kind)] = decodeEntityMap(m)
	}
	if out.Submissions == nil {
		out.Submissions = []Submission{}
	}
	if out.JobInstances == nil {
		out.JobInstances = []JobInstance{}
	}
	return out
}

func encodeEntityMap(m EntityMap) map[string]map[string]any {
	if m == nil {
		return map[string]map[string]any{}
	}
	out := make(map[string]map[string]any, len(m))
	for id, facts := range m {
		out[encodeMongoKey(id)] = encodeStringKeyedMap(facts)
	}
	return out
}

func decodeEntityMap(m map[string]map[string]any) EntityMap {
	if m == nil {
		return EntityMap{}
	}
	out := make(EntityMap, len(m))
	for id, facts := range m {
		out[decodeMongoKey(id)] = decodeStringKeyedMap(facts)
	}
	return out
}

func encodeSubmissions(in []Submission) []mongoSubmissionDoc {
	if in == nil {
		return []mongoSubmissionDoc{}
	}
	out := make([]mongoSubmissionDoc, len(in))
	for i, s := range in {
		out[i] = mongoSubmissionDoc{
			JobID:         s.JobID,
			TaskID:        s.TaskID,
			JobInstanceID: s.JobInstanceID,
			Timestamp:     s.Timestamp,
			UserID:        s.UserID,
			UserName:      s.UserName,
			Facts:         encodeStringKeyedMap(s.Facts),
			IDsTouched:    append([]string(nil), s.IDsTouched...),
		}
	}
	return out
}

func decodeSubmissions(in []mongoSubmissionDoc) []Submission {
	if in == nil {
		return []Submission{}
	}
	out := make([]Submission, len(in))
	for i, s := range in {
		out[i] = Submission{
			JobID:         s.JobID,
			TaskID:        s.TaskID,
			JobInstanceID: s.JobInstanceID,
			Timestamp:     s.Timestamp,
			UserID:        s.UserID,
			UserName:      s.UserName,
			Facts:         decodeStringKeyedMap(s.Facts),
			IDsTouched:    append([]string(nil), s.IDsTouched...),
		}
	}
	return out
}

func encodeStringKeyedMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[encodeMongoKey(k)] = v
	}
	return out
}

func decodeStringKeyedMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[decodeMongoKey(k)] = v
	}
	return out
}

// encodeMongoKey mirrors Process progress key encoding for Mongo field names.
func encodeMongoKey(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}

func decodeMongoKey(key string) string {
	return strings.ReplaceAll(key, "_", ".")
}
