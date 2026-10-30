package living

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionLivingDomains      = "living_domains"
	collectionLivingOrgConfigs   = "living_org_configs"
	collectionLivingActiveConfig = "living_active_config"
)

var (
	_ DomainStore = (*MongoDomainStore)(nil)
	_ ConfigStore = (*MongoConfigStore)(nil)
)

// MongoDomainStore persists Domain documents in MongoDB (one per org).
type MongoDomainStore struct {
	db *mongo.Database
}

func NewMongoDomainStore(db *mongo.Database) *MongoDomainStore {
	return &MongoDomainStore{db: db}
}

type mongoDomainDoc struct {
	Org          string                               `bson:"_id"`
	Load         map[string]map[string]any            `bson:"load"`
	Pallet       map[string]map[string]any            `bson:"pallet"`
	Container    map[string]map[string]any            `bson:"container"`
	Extra        map[string]map[string]map[string]any `bson:"extra,omitempty"`
	Submissions  []mongoSubmissionDoc                 `bson:"submissions"`
	JobInstances []JobInstance                        `bson:"jobInstances"`
}

type mongoSubmissionDoc struct {
	JobID         string         `bson:"jobId"`
	TaskID        string         `bson:"taskId"`
	JobInstanceID string         `bson:"jobInstanceId,omitempty"`
	Timestamp     string         `bson:"timestamp"`
	UserID        string         `bson:"userId"`
	Facts         map[string]any `bson:"facts"`
	IDsTouched    []string       `bson:"idsTouched"`
}

func (s *MongoDomainStore) coll() *mongo.Collection {
	return s.db.Collection(collectionLivingDomains)
}

func (s *MongoDomainStore) Get(ctx context.Context, org string) (Domain, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return EmptyDomain(), ErrNotFound
	}
	var doc mongoDomainDoc
	err := s.coll().FindOne(ctx, bson.M{"_id": org}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return EmptyDomain(), ErrNotFound
	}
	if err != nil {
		return EmptyDomain(), fmt.Errorf("living mongo domain get: %w", err)
	}
	return domainFromMongoDoc(doc), nil
}

func (s *MongoDomainStore) Put(ctx context.Context, org string, domain Domain) error {
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("living mongo domain put: org required")
	}
	doc := domainToMongoDoc(org, domain)
	_, err := s.coll().ReplaceOne(ctx, bson.M{"_id": org}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("living mongo domain put: %w", err)
	}
	return nil
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
	Version string `bson:"version"`
	Payload []byte `bson:"payload"`
}

type mongoActiveConfigDoc struct {
	Org     string `bson:"_id"`
	Version string `bson:"version"`
}

func (s *MongoConfigStore) configs() *mongo.Collection {
	return s.db.Collection(collectionLivingOrgConfigs)
}

func (s *MongoConfigStore) active() *mongo.Collection {
	return s.db.Collection(collectionLivingActiveConfig)
}

func (s *MongoConfigStore) Get(ctx context.Context, org, version string) (OrgConfig, error) {
	org = strings.TrimSpace(org)
	version = strings.TrimSpace(version)
	if org == "" || version == "" {
		return OrgConfig{}, ErrNotFound
	}
	var doc mongoOrgConfigDoc
	err := s.configs().FindOne(ctx, bson.M{"_id": configKey(org, version)}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return OrgConfig{}, ErrNotFound
	}
	if err != nil {
		return OrgConfig{}, fmt.Errorf("living mongo config get: %w", err)
	}
	var cfg OrgConfig
	if err := json.Unmarshal(doc.Payload, &cfg); err != nil {
		return OrgConfig{}, fmt.Errorf("living mongo config decode: %w", err)
	}
	cfg.Org = org
	cfg.Version = version
	return cfg, nil
}

func (s *MongoConfigStore) Put(ctx context.Context, org, version string, cfg OrgConfig) error {
	org = strings.TrimSpace(org)
	version = strings.TrimSpace(version)
	if org == "" || version == "" {
		return fmt.Errorf("living mongo config put: org and version required")
	}
	cfg.Org = org
	cfg.Version = version
	payload, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("living mongo config encode: %w", err)
	}
	doc := mongoOrgConfigDoc{
		ID:      configKey(org, version),
		Org:     org,
		Version: version,
		Payload: payload,
	}
	_, err = s.configs().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("living mongo config put: %w", err)
	}
	return nil
}

func (s *MongoConfigStore) ActiveVersion(ctx context.Context, org string) (string, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return "", ErrNotFound
	}
	var doc mongoActiveConfigDoc
	err := s.active().FindOne(ctx, bson.M{"_id": org}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("living mongo active get: %w", err)
	}
	if strings.TrimSpace(doc.Version) == "" {
		return "", ErrNotFound
	}
	return doc.Version, nil
}

func (s *MongoConfigStore) SetActiveVersion(ctx context.Context, org, version string) error {
	org = strings.TrimSpace(org)
	version = strings.TrimSpace(version)
	if org == "" || version == "" {
		return fmt.Errorf("living mongo active set: org and version required")
	}
	err := s.configs().FindOne(ctx, bson.M{"_id": configKey(org, version)}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("living mongo active set: lookup: %w", err)
	}
	doc := mongoActiveConfigDoc{Org: org, Version: version}
	_, err = s.active().ReplaceOne(ctx, bson.M{"_id": org}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("living mongo active set: %w", err)
	}
	return nil
}

func domainToMongoDoc(org string, domain Domain) mongoDomainDoc {
	doc := mongoDomainDoc{
		Org:          org,
		Load:         encodeEntityMap(domain.Load),
		Pallet:       encodeEntityMap(domain.Pallet),
		Container:    encodeEntityMap(domain.Container),
		Submissions:  encodeSubmissions(domain.Submissions),
		JobInstances: append([]JobInstance(nil), domain.JobInstances...),
	}
	if len(domain.Extra) > 0 {
		doc.Extra = make(map[string]map[string]map[string]any, len(domain.Extra))
		for kind, m := range domain.Extra {
			encKind := encodeMongoKey(kind)
			if m == nil {
				doc.Extra[encKind] = map[string]map[string]any{}
				continue
			}
			doc.Extra[encKind] = encodeEntityMap(*m)
		}
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
	out := Domain{
		Load:         decodeEntityMap(doc.Load),
		Pallet:       decodeEntityMap(doc.Pallet),
		Container:    decodeEntityMap(doc.Container),
		Submissions:  decodeSubmissions(doc.Submissions),
		JobInstances: append([]JobInstance(nil), doc.JobInstances...),
	}
	if len(doc.Extra) > 0 {
		out.Extra = make(map[string]*EntityMap, len(doc.Extra))
		for kind, m := range doc.Extra {
			decoded := decodeEntityMap(m)
			out.Extra[decodeMongoKey(kind)] = &decoded
		}
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
