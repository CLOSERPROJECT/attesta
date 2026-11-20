package org

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const collectionOrgDPPs = "org_dpps"

var _ DPPStore = (*MongoDPPStore)(nil)

// MongoDPPStore persists DPPs in MongoDB (one doc per org+stream+subject).
type MongoDPPStore struct {
	db *mongo.Database
}

func NewMongoDPPStore(db *mongo.Database) *MongoDPPStore {
	return &MongoDPPStore{db: db}
}

type mongoDPPDoc struct {
	ID            string         `bson:"_id"`
	Org           string         `bson:"org"`
	Stream        string         `bson:"stream"`
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

func (s *MongoDPPStore) coll() *mongo.Collection {
	return s.db.Collection(collectionOrgDPPs)
}

func (s *MongoDPPStore) Get(ctx context.Context, scope Scope, subjectID string) (DPPRecord, error) {
	key, err := passportStoreKey(scope, subjectID)
	if err != nil {
		return DPPRecord{}, ErrNotFound
	}
	var doc mongoDPPDoc
	err = s.coll().FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return DPPRecord{}, ErrNotFound
	}
	if err != nil {
		return DPPRecord{}, fmt.Errorf("org mongo passport get: %w", err)
	}
	return dppFromMongoDoc(doc), nil
}

func (s *MongoDPPStore) GetByIdentity(ctx context.Context, gtin, lot, serial string) (DPPRecord, error) {
	gtin = strings.TrimSpace(gtin)
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if gtin == "" || lot == "" || serial == "" {
		return DPPRecord{}, ErrNotFound
	}
	var doc mongoDPPDoc
	err := s.coll().FindOne(ctx, bson.M{"gtin": gtin, "lot": lot, "serial": serial}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return DPPRecord{}, ErrNotFound
	}
	if err != nil {
		return DPPRecord{}, fmt.Errorf("org mongo passport get by identity: %w", err)
	}
	return dppFromMongoDoc(doc), nil
}

func (s *MongoDPPStore) Create(ctx context.Context, record DPPRecord) error {
	scope, err := Scope{Org: record.Org, Stream: record.Stream}.Normalize()
	if err != nil {
		return err
	}
	key, err := passportStoreKey(scope, record.SubjectID)
	if err != nil {
		return err
	}
	gtin := strings.TrimSpace(record.GTIN)
	lot := strings.TrimSpace(record.Lot)
	serial := strings.TrimSpace(record.Serial)
	if gtin != "" && lot != "" && serial != "" {
		if _, err := s.GetByIdentity(ctx, gtin, lot, serial); err == nil {
			return ErrDPPExists
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	doc := mongoDPPDoc{
		ID:            key,
		Org:           scope.Org,
		Stream:        scope.Stream,
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
			return ErrDPPExists
		}
		return fmt.Errorf("org mongo passport create: %w", err)
	}
	return nil
}

func (s *MongoDPPStore) DeleteByOrg(ctx context.Context, org string) error {
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("org: passport delete by org requires org")
	}
	_, err := s.coll().DeleteMany(ctx, bson.M{"org": org})
	if err != nil {
		return fmt.Errorf("org mongo passport delete by org: %w", err)
	}
	return nil
}

func dppFromMongoDoc(doc mongoDPPDoc) DPPRecord {
	return DPPRecord{
		Org:           doc.Org,
		Stream:        doc.Stream,
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
