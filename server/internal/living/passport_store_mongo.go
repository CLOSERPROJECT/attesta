package living

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const collectionLivingPassports = "living_passports"

var _ PassportStore = (*MongoPassportStore)(nil)

// MongoPassportStore persists Passports in MongoDB (one doc per org+subject).
type MongoPassportStore struct {
	db *mongo.Database
}

func NewMongoPassportStore(db *mongo.Database) *MongoPassportStore {
	return &MongoPassportStore{db: db}
}

type mongoPassportDoc struct {
	ID            string         `bson:"_id"`
	Org           string         `bson:"org"`
	SubjectID     string         `bson:"subjectId"`
	GTIN          string         `bson:"gtin"`
	Lot           string         `bson:"lot"`
	Serial        string         `bson:"serial"`
	ProductName   string         `bson:"productName,omitempty"`
	Description   string         `bson:"description,omitempty"`
	Snapshot      map[string]any `bson:"snapshot"`
	ConfigVersion string         `bson:"configVersion,omitempty"`
	IssuedAt      string         `bson:"issuedAt"`
	DigitalLink   string         `bson:"digitalLink"`
}

func (s *MongoPassportStore) coll() *mongo.Collection {
	return s.db.Collection(collectionLivingPassports)
}

func (s *MongoPassportStore) Get(ctx context.Context, org, subjectID string) (PassportRecord, error) {
	key, err := passportStoreKey(org, subjectID)
	if err != nil {
		return PassportRecord{}, ErrNotFound
	}
	var doc mongoPassportDoc
	err = s.coll().FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return PassportRecord{}, ErrNotFound
	}
	if err != nil {
		return PassportRecord{}, fmt.Errorf("living mongo passport get: %w", err)
	}
	return passportFromMongoDoc(doc), nil
}

func (s *MongoPassportStore) GetByIdentity(ctx context.Context, gtin, lot, serial string) (PassportRecord, error) {
	gtin = strings.TrimSpace(gtin)
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if gtin == "" || lot == "" || serial == "" {
		return PassportRecord{}, ErrNotFound
	}
	var doc mongoPassportDoc
	err := s.coll().FindOne(ctx, bson.M{"gtin": gtin, "lot": lot, "serial": serial}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return PassportRecord{}, ErrNotFound
	}
	if err != nil {
		return PassportRecord{}, fmt.Errorf("living mongo passport get by identity: %w", err)
	}
	return passportFromMongoDoc(doc), nil
}

func (s *MongoPassportStore) Create(ctx context.Context, record PassportRecord) error {
	key, err := passportStoreKey(record.Org, record.SubjectID)
	if err != nil {
		return err
	}
	doc := mongoPassportDoc{
		ID:            key,
		Org:           strings.TrimSpace(record.Org),
		SubjectID:     strings.TrimSpace(record.SubjectID),
		GTIN:          record.GTIN,
		Lot:           record.Lot,
		Serial:        record.Serial,
		ProductName:   record.ProductName,
		Description:   record.Description,
		Snapshot:      cloneAnyMap(record.Snapshot),
		ConfigVersion: record.ConfigVersion,
		IssuedAt:      record.IssuedAt,
		DigitalLink:   record.DigitalLink,
	}
	if doc.Snapshot == nil {
		doc.Snapshot = map[string]any{}
	}
	_, err = s.coll().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrPassportExists
		}
		return fmt.Errorf("living mongo passport create: %w", err)
	}
	return nil
}

func (s *MongoPassportStore) DeleteByOrg(ctx context.Context, org string) error {
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("living: passport delete by org requires org")
	}
	_, err := s.coll().DeleteMany(ctx, bson.M{"org": org})
	if err != nil {
		return fmt.Errorf("living mongo passport delete by org: %w", err)
	}
	return nil
}

func passportFromMongoDoc(doc mongoPassportDoc) PassportRecord {
	return PassportRecord{
		Org:           doc.Org,
		SubjectID:     doc.SubjectID,
		GTIN:          doc.GTIN,
		Lot:           doc.Lot,
		Serial:        doc.Serial,
		ProductName:   doc.ProductName,
		Description:   doc.Description,
		Snapshot:      cloneAnyMap(doc.Snapshot),
		ConfigVersion: doc.ConfigVersion,
		IssuedAt:      doc.IssuedAt,
		DigitalLink:   doc.DigitalLink,
	}
}
