package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store interface {
	InsertProcess(ctx context.Context, process Process) (primitive.ObjectID, error)
	LoadProcessByID(ctx context.Context, id primitive.ObjectID) (*Process, error)
	LoadLatestProcessByWorkflow(ctx context.Context, workflowKey string) (*Process, error)
	LoadProcessByDigitalLink(ctx context.Context, gtin, lot, serial string) (*Process, error)
	ListRecentProcessesByWorkflow(ctx context.Context, workflowKey string, limit int64) ([]Process, error)
	HasProcessesByWorkflow(ctx context.Context, workflowKey string) (bool, error)
	UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, workflowKey, substepID string, progress ProcessStep) error
	UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, workflowKey, status string) error
	UpdateProcessDPP(ctx context.Context, id primitive.ObjectID, workflowKey string, dpp ProcessDPP) error
	InsertNotarization(ctx context.Context, notarization Notarization) error
	SaveAttachment(ctx context.Context, upload AttachmentUpload, content io.Reader) (Attachment, error)
	LoadAttachmentByID(ctx context.Context, id primitive.ObjectID) (*Attachment, error)
	OpenAttachmentDownload(ctx context.Context, id primitive.ObjectID) (io.ReadCloser, error)
	SaveFormataBuilderStream(ctx context.Context, stream FormataBuilderStream) (FormataBuilderStream, error)
	UpdateFormataBuilderStream(ctx context.Context, stream FormataBuilderStream) (FormataBuilderStream, error)
	LoadFormataBuilderStream(ctx context.Context) (*FormataBuilderStream, error)
	LoadFormataBuilderStreamByID(ctx context.Context, id primitive.ObjectID) (*FormataBuilderStream, error)
	ListFormataBuilderStreams(ctx context.Context) ([]FormataBuilderStream, error)
	DeleteFormataBuilderStream(ctx context.Context, id primitive.ObjectID) error
	DeleteWorkflowData(ctx context.Context, workflowKey string) error
}

type Organization struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Slug             string             `bson:"slug"`
	Name             string             `bson:"name"`
	LogoAttachmentID string             `bson:"logoAttachmentId,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt"`
}

type Role struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	OrgID     primitive.ObjectID `bson:"orgId"`
	OrgSlug   string             `bson:"orgSlug"`
	Slug      string             `bson:"slug"`
	Name      string             `bson:"name"`
	Color     string             `bson:"color,omitempty"`
	Border    string             `bson:"border,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
}

type AccountUser struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty"`
	IdentityUserID  string              `bson:"identityUserId,omitempty"`
	OrgID           *primitive.ObjectID `bson:"orgId,omitempty"`
	OrgSlug         string              `bson:"orgSlug,omitempty"`
	Email           string              `bson:"email"`
	PasswordHash    string              `bson:"passwordHash"`
	RoleSlugs       []string            `bson:"roleSlugs"`
	Status          string              `bson:"status"`
	IsPlatformAdmin bool                `bson:"isPlatformAdmin,omitempty"`
	CreatedAt       time.Time           `bson:"createdAt"`
	LastLoginAt     *time.Time          `bson:"lastLoginAt,omitempty"`
}

type FormataBuilderStream struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Stream          string             `bson:"stream"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
	CreatedByUserID string             `bson:"createdByUserId,omitempty"`
	UpdatedByUserID string             `bson:"updatedByUserId"`
}

const (
	collectionFormataStream = "formata_builder_streams"
)

type MongoStore struct {
	db     *mongo.Database
	dbPort mongoDatabasePort
}

type mongoDatabasePort interface {
	Collection(name string) mongoCollectionPort
	NewGridFSBucket(name string) (gridFSBucketPort, error)
}

type mongoCollectionPort interface {
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort
	CreateIndexes(ctx context.Context, models []mongo.IndexModel) error
	DropIndex(ctx context.Context, name string) error
}

type mongoSingleResultPort interface {
	Decode(v interface{}) error
	Err() error
}

type mongoCursorPort interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Close(ctx context.Context) error
}

type gridFSBucketPort interface {
	UploadFromStreamWithID(id interface{}, filename string, source io.Reader, opts ...*options.UploadOptions) error
	OpenDownloadStream(fileID interface{}) (io.ReadCloser, error)
	Delete(fileID interface{}) error
}

type mongoDriverDatabase struct {
	db *mongo.Database
}

func (d mongoDriverDatabase) Collection(name string) mongoCollectionPort {
	return mongoDriverCollection{collection: d.db.Collection(name)}
}

func (d mongoDriverDatabase) NewGridFSBucket(name string) (gridFSBucketPort, error) {
	bucket, err := gridfs.NewBucket(d.db, options.GridFSBucket().SetName(name))
	if err != nil {
		return nil, err
	}
	return mongoDriverGridFSBucket{bucket: bucket}, nil
}

type mongoDriverCollection struct {
	collection *mongo.Collection
}

func (c mongoDriverCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return c.collection.InsertOne(ctx, document, opts...)
}

func (c mongoDriverCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
	return mongoDriverSingleResult{result: c.collection.FindOne(ctx, filter, opts...)}
}

func (c mongoDriverCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
	cursor, err := c.collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return mongoDriverCursor{cursor: cursor}, nil
}

func (c mongoDriverCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return c.collection.UpdateOne(ctx, filter, update, opts...)
}

func (c mongoDriverCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return c.collection.DeleteOne(ctx, filter, opts...)
}

func (c mongoDriverCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return c.collection.DeleteMany(ctx, filter, opts...)
}

func (c mongoDriverCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
	return mongoDriverSingleResult{result: c.collection.FindOneAndUpdate(ctx, filter, update, opts...)}
}

func (c mongoDriverCollection) CreateIndexes(ctx context.Context, models []mongo.IndexModel) error {
	_, err := c.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (c mongoDriverCollection) DropIndex(ctx context.Context, name string) error {
	_, err := c.collection.Indexes().DropOne(ctx, name)
	if err != nil && isMongoIndexNotFoundError(err) {
		return nil
	}
	return err
}

func isMongoIndexNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var commandErr mongo.CommandError
	if errors.As(err, &commandErr) {
		return commandErr.Code == 26 || commandErr.Code == 27
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "index not found") || strings.Contains(msg, "ns not found")
}

type mongoDriverSingleResult struct {
	result *mongo.SingleResult
}

func (r mongoDriverSingleResult) Decode(v interface{}) error {
	return r.result.Decode(v)
}

func (r mongoDriverSingleResult) Err() error {
	return r.result.Err()
}

type mongoDriverCursor struct {
	cursor *mongo.Cursor
}

func (c mongoDriverCursor) Next(ctx context.Context) bool {
	return c.cursor.Next(ctx)
}

func (c mongoDriverCursor) Decode(val interface{}) error {
	return c.cursor.Decode(val)
}

func (c mongoDriverCursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}

type mongoDriverGridFSBucket struct {
	bucket *gridfs.Bucket
}

func (b mongoDriverGridFSBucket) UploadFromStreamWithID(id interface{}, filename string, source io.Reader, opts ...*options.UploadOptions) error {
	return b.bucket.UploadFromStreamWithID(id, filename, source, opts...)
}

func (b mongoDriverGridFSBucket) OpenDownloadStream(fileID interface{}) (io.ReadCloser, error) {
	return b.bucket.OpenDownloadStream(fileID)
}

func (b mongoDriverGridFSBucket) Delete(fileID interface{}) error {
	return b.bucket.Delete(fileID)
}

func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{
		db:     db,
		dbPort: mongoDriverDatabase{db: db},
	}
}

func (s *MongoStore) database() mongoDatabasePort {
	if s.dbPort != nil {
		return s.dbPort
	}
	if s.db == nil {
		return nil
	}
	s.dbPort = mongoDriverDatabase{db: s.db}
	return s.dbPort
}

var ErrAttachmentTooLarge = errors.New("attachment too large")

type Attachment struct {
	ID          primitive.ObjectID
	ProcessID   primitive.ObjectID
	SubstepID   string
	Filename    string
	ContentType string
	SizeBytes   int64
	SHA256      string
	UploadedAt  time.Time
}

type AttachmentUpload struct {
	ProcessID   primitive.ObjectID
	SubstepID   string
	Filename    string
	ContentType string
	MaxBytes    int64
	UploadedAt  time.Time
}

func (s *MongoStore) InsertProcess(ctx context.Context, process Process) (primitive.ObjectID, error) {
	result, err := s.database().Collection("processes").InsertOne(ctx, process)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("invalid inserted id")
	}
	return id, nil
}

func (s *MongoStore) LoadProcessByID(ctx context.Context, id primitive.ObjectID) (*Process, error) {
	var process Process
	if err := s.database().Collection("processes").FindOne(ctx, bson.M{"_id": id}).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) LoadLatestProcessByWorkflow(ctx context.Context, workflowKey string) (*Process, error) {
	filter := bson.M{"workflowKey": workflowKey}
	if workflowKey == "workflow" {
		filter = bson.M{"$or": []bson.M{{"workflowKey": workflowKey}, {"workflowKey": bson.M{"$exists": false}}}}
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var process Process
	if err := s.database().Collection("processes").FindOne(ctx, filter, opts).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) ListRecentProcessesByWorkflow(ctx context.Context, workflowKey string, limit int64) ([]Process, error) {
	filter := bson.M{"workflowKey": workflowKey}
	if workflowKey == "workflow" {
		filter = bson.M{"$or": []bson.M{{"workflowKey": workflowKey}, {"workflowKey": bson.M{"$exists": false}}}}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cursor, err := s.database().Collection("processes").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var processes []Process
	for cursor.Next(ctx) {
		var process Process
		if err := cursor.Decode(&process); err != nil {
			continue
		}
		processes = append(processes, process)
	}
	return processes, nil
}

func (s *MongoStore) HasProcessesByWorkflow(ctx context.Context, workflowKey string) (bool, error) {
	err := s.database().Collection("processes").FindOne(
		ctx,
		bson.M{"workflowKey": strings.TrimSpace(workflowKey)},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Err()
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, mongo.ErrNoDocuments):
		return false, nil
	default:
		return false, err
	}
}

func (s *MongoStore) LoadProcessByDigitalLink(ctx context.Context, gtin, lot, serial string) (*Process, error) {
	filter := bson.M{
		"dpp.gtin":   strings.TrimSpace(gtin),
		"dpp.lot":    strings.TrimSpace(lot),
		"dpp.serial": strings.TrimSpace(serial),
	}
	var process Process
	if err := s.database().Collection("processes").FindOne(ctx, filter).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, workflowKey, substepID string, progress ProcessStep) error {
	update := bson.M{
		"$set": bson.M{
			"workflowKey": workflowKey,
			"progress." + encodeProgressKey(substepID): progress,
		},
	}
	return s.database().Collection("processes").FindOneAndUpdate(ctx, bson.M{"_id": id}, update).Err()
}

func (s *MongoStore) UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, workflowKey, status string) error {
	_, err := s.database().Collection("processes").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status, "workflowKey": workflowKey}})
	return err
}

func (s *MongoStore) UpdateProcessDPP(ctx context.Context, id primitive.ObjectID, workflowKey string, dpp ProcessDPP) error {
	update := bson.M{
		"$set": bson.M{
			"workflowKey": workflowKey,
			"dpp":         dpp,
		},
	}
	_, err := s.database().Collection("processes").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (s *MongoStore) InsertNotarization(ctx context.Context, notarization Notarization) error {
	_, err := s.database().Collection("notarizations").InsertOne(ctx, notarization)
	return err
}

func (s *MongoStore) SaveAttachment(ctx context.Context, upload AttachmentUpload, content io.Reader) (Attachment, error) {
	bucket, err := s.attachmentsBucket()
	if err != nil {
		return Attachment{}, err
	}

	filename := strings.TrimSpace(upload.Filename)
	if filename == "" {
		filename = "attachment"
	}
	contentType := strings.TrimSpace(upload.ContentType)
	if contentType == "" {
		contentType = detectAttachmentContentType(filename)
	}

	uploadedAt := upload.UploadedAt
	if uploadedAt.IsZero() {
		uploadedAt = time.Now().UTC()
	}

	id := primitive.NewObjectID()
	tracker := newAttachmentTracker(upload.MaxBytes)
	reader := io.TeeReader(content, tracker)
	uploadOpts := options.GridFSUpload().SetMetadata(bson.M{
		"processId":   upload.ProcessID,
		"substepId":   upload.SubstepID,
		"contentType": contentType,
		"uploadedAt":  uploadedAt,
	})
	if err := bucket.UploadFromStreamWithID(id, filename, reader, uploadOpts); err != nil {
		if errors.Is(err, ErrAttachmentTooLarge) {
			_ = bucket.Delete(id)
			return Attachment{}, ErrAttachmentTooLarge
		}
		return Attachment{}, err
	}
	sha := tracker.SHA256()
	if _, err := s.database().Collection("attachments.files").UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"metadata.sha256": sha}},
	); err != nil {
		return Attachment{}, err
	}

	return Attachment{
		ID:          id,
		ProcessID:   upload.ProcessID,
		SubstepID:   upload.SubstepID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   tracker.Size(),
		SHA256:      sha,
		UploadedAt:  uploadedAt,
	}, nil
}

func (s *MongoStore) LoadAttachmentByID(ctx context.Context, id primitive.ObjectID) (*Attachment, error) {
	var doc struct {
		ID         primitive.ObjectID `bson:"_id"`
		Filename   string             `bson:"filename"`
		Length     int64              `bson:"length"`
		UploadDate time.Time          `bson:"uploadDate"`
		Metadata   struct {
			ProcessID   primitive.ObjectID `bson:"processId"`
			SubstepID   string             `bson:"substepId"`
			ContentType string             `bson:"contentType"`
			UploadedAt  time.Time          `bson:"uploadedAt"`
			SHA256      string             `bson:"sha256"`
		} `bson:"metadata"`
	}
	if err := s.database().Collection("attachments.files").FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		return nil, err
	}
	uploadedAt := doc.Metadata.UploadedAt
	if uploadedAt.IsZero() {
		uploadedAt = doc.UploadDate
	}
	attachment := &Attachment{
		ID:          doc.ID,
		ProcessID:   doc.Metadata.ProcessID,
		SubstepID:   doc.Metadata.SubstepID,
		Filename:    doc.Filename,
		ContentType: doc.Metadata.ContentType,
		SizeBytes:   doc.Length,
		SHA256:      doc.Metadata.SHA256,
		UploadedAt:  uploadedAt,
	}
	return attachment, nil
}

func (s *MongoStore) OpenAttachmentDownload(ctx context.Context, id primitive.ObjectID) (io.ReadCloser, error) {
	bucket, err := s.attachmentsBucket()
	if err != nil {
		return nil, err
	}
	return bucket.OpenDownloadStream(id)
}

func (s *MongoStore) attachmentsBucket() (gridFSBucketPort, error) {
	return s.database().NewGridFSBucket("attachments")
}

type MemoryStore struct {
	mu             sync.RWMutex
	processes      map[primitive.ObjectID]Process
	notarizations  []Notarization
	attachments    map[primitive.ObjectID]memoryAttachment
	formataStreams map[primitive.ObjectID]FormataBuilderStream

	InsertProcessErr  error
	LoadProcessErr    error
	LoadLatestErr     error
	ListProcessesErr  error
	UpdateProgressErr error
	UpdateStatusErr   error
	InsertNotarizeErr error
}

type memoryAttachment struct {
	meta    Attachment
	content []byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		processes:      map[primitive.ObjectID]Process{},
		attachments:    map[primitive.ObjectID]memoryAttachment{},
		formataStreams: map[primitive.ObjectID]FormataBuilderStream{},
	}
}

func (s *MemoryStore) SeedProcess(process Process) primitive.ObjectID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if process.ID.IsZero() {
		process.ID = primitive.NewObjectID()
	}
	s.processes[process.ID] = cloneProcess(process)
	return process.ID
}

func (s *MemoryStore) SnapshotProcess(id primitive.ObjectID) (Process, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	process, ok := s.processes[id]
	if !ok {
		return Process{}, false
	}
	return cloneProcess(process), true
}

func (s *MemoryStore) Notarizations() []Notarization {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Notarization, len(s.notarizations))
	copy(items, s.notarizations)
	return items
}

func (s *MemoryStore) InsertProcess(_ context.Context, process Process) (primitive.ObjectID, error) {
	if s.InsertProcessErr != nil {
		return primitive.NilObjectID, s.InsertProcessErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if process.ID.IsZero() {
		process.ID = primitive.NewObjectID()
	}
	s.processes[process.ID] = cloneProcess(process)
	return process.ID, nil
}

func (s *MemoryStore) LoadProcessByID(_ context.Context, id primitive.ObjectID) (*Process, error) {
	if s.LoadProcessErr != nil {
		return nil, s.LoadProcessErr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	process, ok := s.processes[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneProcess(process)
	return &cloned, nil
}

func (s *MemoryStore) LoadLatestProcessByWorkflow(_ context.Context, workflowKey string) (*Process, error) {
	if s.LoadLatestErr != nil {
		return nil, s.LoadLatestErr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.processes) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	var latest Process
	first := true
	for _, process := range s.processes {
		key := strings.TrimSpace(process.WorkflowKey)
		if key != workflowKey {
			if !(workflowKey == "workflow" && key == "") {
				continue
			}
		}
		if first || process.CreatedAt.After(latest.CreatedAt) {
			latest = process
			first = false
		}
	}
	if first {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneProcess(latest)
	return &cloned, nil
}

func (s *MemoryStore) ListRecentProcessesByWorkflow(_ context.Context, workflowKey string, limit int64) ([]Process, error) {
	if s.ListProcessesErr != nil {
		return nil, s.ListProcessesErr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Process, 0, len(s.processes))
	for _, process := range s.processes {
		key := strings.TrimSpace(process.WorkflowKey)
		if key != workflowKey {
			if !(workflowKey == "workflow" && key == "") {
				continue
			}
		}
		items = append(items, cloneProcess(process))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && int64(len(items)) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *MemoryStore) HasProcessesByWorkflow(_ context.Context, workflowKey string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, process := range s.processes {
		if strings.TrimSpace(process.WorkflowKey) == strings.TrimSpace(workflowKey) {
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) UpdateProcessProgress(_ context.Context, id primitive.ObjectID, workflowKey, substepID string, progress ProcessStep) error {
	if s.UpdateProgressErr != nil {
		return s.UpdateProgressErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	if !ok {
		return mongo.ErrNoDocuments
	}
	if process.Progress == nil {
		process.Progress = map[string]ProcessStep{}
	}
	process.WorkflowKey = strings.TrimSpace(workflowKey)
	process.Progress[encodeProgressKey(substepID)] = cloneProcessStep(progress)
	s.processes[id] = process
	return nil
}

func (s *MemoryStore) UpdateProcessStatus(_ context.Context, id primitive.ObjectID, workflowKey, status string) error {
	if s.UpdateStatusErr != nil {
		return s.UpdateStatusErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	if !ok {
		return mongo.ErrNoDocuments
	}
	process.WorkflowKey = strings.TrimSpace(workflowKey)
	process.Status = status
	s.processes[id] = process
	return nil
}

func (s *MemoryStore) UpdateProcessDPP(_ context.Context, id primitive.ObjectID, workflowKey string, dpp ProcessDPP) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	if !ok {
		return mongo.ErrNoDocuments
	}
	process.WorkflowKey = strings.TrimSpace(workflowKey)
	dppCopy := dpp
	process.DPP = &dppCopy
	s.processes[id] = process
	return nil
}

func (s *MemoryStore) LoadProcessByDigitalLink(_ context.Context, gtin, lot, serial string) (*Process, error) {
	trimGTIN := strings.TrimSpace(gtin)
	trimLot := strings.TrimSpace(lot)
	trimSerial := strings.TrimSpace(serial)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, process := range s.processes {
		if process.DPP == nil {
			continue
		}
		if process.DPP.GTIN == trimGTIN && process.DPP.Lot == trimLot && process.DPP.Serial == trimSerial {
			cloned := cloneProcess(process)
			return &cloned, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}

func (s *MemoryStore) InsertNotarization(_ context.Context, notarization Notarization) error {
	if s.InsertNotarizeErr != nil {
		return s.InsertNotarizeErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if notarization.ID.IsZero() {
		notarization.ID = primitive.NewObjectID()
	}
	s.notarizations = append(s.notarizations, notarization)
	return nil
}

func (s *MemoryStore) SaveAttachment(_ context.Context, upload AttachmentUpload, content io.Reader) (Attachment, error) {
	filename := strings.TrimSpace(upload.Filename)
	if filename == "" {
		filename = "attachment"
	}
	contentType := strings.TrimSpace(upload.ContentType)
	if contentType == "" {
		contentType = detectAttachmentContentType(filename)
	}

	uploadedAt := upload.UploadedAt
	if uploadedAt.IsZero() {
		uploadedAt = time.Now().UTC()
	}

	var body bytes.Buffer
	tracker := newAttachmentTracker(upload.MaxBytes)
	reader := io.TeeReader(content, tracker)
	if _, err := io.Copy(&body, reader); err != nil {
		if errors.Is(err, ErrAttachmentTooLarge) {
			return Attachment{}, ErrAttachmentTooLarge
		}
		return Attachment{}, err
	}

	attachment := Attachment{
		ID:          primitive.NewObjectID(),
		ProcessID:   upload.ProcessID,
		SubstepID:   upload.SubstepID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   tracker.Size(),
		SHA256:      tracker.SHA256(),
		UploadedAt:  uploadedAt,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.attachments[attachment.ID] = memoryAttachment{
		meta:    attachment,
		content: body.Bytes(),
	}
	return attachment, nil
}

func (s *MemoryStore) LoadAttachmentByID(_ context.Context, id primitive.ObjectID) (*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.attachments[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	attachment := item.meta
	return &attachment, nil
}

func (s *MemoryStore) OpenAttachmentDownload(_ context.Context, id primitive.ObjectID) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.attachments[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	content := append([]byte(nil), item.content...)
	return io.NopCloser(bytes.NewReader(content)), nil
}

func (s *MemoryStore) SaveFormataBuilderStream(_ context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stream.ID.IsZero() {
		stream.ID = primitive.NewObjectID()
	}
	if stream.UpdatedAt.IsZero() {
		stream.UpdatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(stream.CreatedByUserID) == "" {
		stream.CreatedByUserID = strings.TrimSpace(stream.UpdatedByUserID)
	}
	if _, exists := s.formataStreams[stream.ID]; exists {
		return FormataBuilderStream{}, errors.New("formata builder stream id already exists")
	}
	s.formataStreams[stream.ID] = stream
	return stream, nil
}

func (s *MemoryStore) UpdateFormataBuilderStream(_ context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stream.ID.IsZero() {
		return FormataBuilderStream{}, mongo.ErrNoDocuments
	}
	if _, exists := s.formataStreams[stream.ID]; !exists {
		return FormataBuilderStream{}, mongo.ErrNoDocuments
	}
	if stream.UpdatedAt.IsZero() {
		stream.UpdatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(stream.CreatedByUserID) == "" {
		stream.CreatedByUserID = strings.TrimSpace(stream.UpdatedByUserID)
	}
	s.formataStreams[stream.ID] = stream
	return stream, nil
}

func (s *MemoryStore) LoadFormataBuilderStream(_ context.Context) (*FormataBuilderStream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.formataStreams) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	var latest FormataBuilderStream
	first := true
	for _, stream := range s.formataStreams {
		if first ||
			stream.UpdatedAt.After(latest.UpdatedAt) ||
			(stream.UpdatedAt.Equal(latest.UpdatedAt) && stream.ID.Timestamp().After(latest.ID.Timestamp())) {
			latest = stream
			first = false
		}
	}
	copied := latest
	return &copied, nil
}

func (s *MemoryStore) LoadFormataBuilderStreamByID(_ context.Context, id primitive.ObjectID) (*FormataBuilderStream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stream, ok := s.formataStreams[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copied := stream
	return &copied, nil
}

func (s *MemoryStore) ListFormataBuilderStreams(_ context.Context) ([]FormataBuilderStream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.formataStreams) == 0 {
		return nil, nil
	}
	items := make([]FormataBuilderStream, 0, len(s.formataStreams))
	for _, stream := range s.formataStreams {
		items = append(items, stream)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID.Hex() < items[j].ID.Hex()
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (s *MemoryStore) DeleteFormataBuilderStream(_ context.Context, id primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.formataStreams[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.formataStreams, id)
	return nil
}

func (s *MemoryStore) DeleteWorkflowData(_ context.Context, workflowKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedKey := strings.TrimSpace(workflowKey)
	processIDs := make(map[primitive.ObjectID]struct{})
	for id, process := range s.processes {
		if strings.TrimSpace(process.WorkflowKey) != trimmedKey {
			continue
		}
		processIDs[id] = struct{}{}
		delete(s.processes, id)
	}

	if len(processIDs) == 0 {
		return nil
	}

	notarizations := s.notarizations[:0]
	for _, notarization := range s.notarizations {
		if _, ok := processIDs[notarization.ProcessID]; ok {
			continue
		}
		notarizations = append(notarizations, notarization)
	}
	s.notarizations = notarizations

	for id, attachment := range s.attachments {
		if _, ok := processIDs[attachment.meta.ProcessID]; ok {
			delete(s.attachments, id)
		}
	}

	return nil
}

func cloneProcess(process Process) Process {
	cloned := process
	if process.DPP != nil {
		dpp := *process.DPP
		cloned.DPP = &dpp
	}
	cloned.Progress = make(map[string]ProcessStep, len(process.Progress))
	for key, value := range process.Progress {
		cloned.Progress[key] = cloneProcessStep(value)
	}
	return cloned
}

func cloneProcessStep(step ProcessStep) ProcessStep {
	cloned := step
	if step.DoneAt != nil {
		timestamp := *step.DoneAt
		cloned.DoneAt = &timestamp
	}
	if step.DoneBy != nil {
		actor := *step.DoneBy
		cloned.DoneBy = &actor
	}
	if step.Data != nil {
		cloned.Data = make(map[string]interface{}, len(step.Data))
		for key, value := range step.Data {
			cloned.Data[key] = value
		}
	}
	return cloned
}

type attachmentTracker struct {
	maxBytes int64
	size     int64
	hasher   hashWriter
}

type hashWriter interface {
	Write(p []byte) (int, error)
	Sum(b []byte) []byte
}

func newAttachmentTracker(maxBytes int64) *attachmentTracker {
	return &attachmentTracker{
		maxBytes: maxBytes,
		hasher:   sha256.New(),
	}
}

func (t *attachmentTracker) Write(p []byte) (int, error) {
	next := t.size + int64(len(p))
	if t.maxBytes > 0 && next > t.maxBytes {
		return 0, ErrAttachmentTooLarge
	}
	t.size = next
	if _, err := t.hasher.Write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *attachmentTracker) Size() int64 {
	return t.size
}

func (t *attachmentTracker) SHA256() string {
	return hex.EncodeToString(t.hasher.Sum(nil))
}

func detectAttachmentContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if strings.TrimSpace(mimeType) == "" {
		return "application/octet-stream"
	}
	return mimeType
}

var nonSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

const maxIdentityRoleNameLen = 35
const invalidRoleNameMessage = "role name must contain at least one letter or number"

var nonIdentityRoleSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func canonifyIdentityRoleSlug(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = nonIdentityRoleSlugPattern.ReplaceAllString(normalized, "")
	if len(normalized) > maxIdentityRoleNameLen {
		normalized = normalized[:maxIdentityRoleNameLen]
	}
	return normalized
}

func canonifySlug(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = nonSlugPattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "item"
	}
	return normalized
}

func canonifyRoleSlugs(roleSlugs []string) []string {
	canon := make([]string, 0, len(roleSlugs))
	seen := map[string]struct{}{}
	for _, roleSlug := range roleSlugs {
		slug := canonifySlug(roleSlug)
		if slug == "" {
			continue
		}
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}
		canon = append(canon, slug)
	}
	return canon
}

func canonifyOptionalSlug(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	return canonifySlug(trimmed)
}

func hashLookupToken(token string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(hash[:])
}

func (s *MongoStore) SaveFormataBuilderStream(ctx context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	if stream.ID.IsZero() {
		stream.ID = primitive.NewObjectID()
	}
	if stream.UpdatedAt.IsZero() {
		stream.UpdatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(stream.CreatedByUserID) == "" {
		stream.CreatedByUserID = strings.TrimSpace(stream.UpdatedByUserID)
	}
	if _, err := s.database().Collection(collectionFormataStream).InsertOne(ctx, stream); err != nil {
		return FormataBuilderStream{}, err
	}
	return stream, nil
}

func (s *MongoStore) UpdateFormataBuilderStream(ctx context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	if stream.ID.IsZero() {
		return FormataBuilderStream{}, mongo.ErrNoDocuments
	}
	if stream.UpdatedAt.IsZero() {
		stream.UpdatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(stream.CreatedByUserID) == "" {
		stream.CreatedByUserID = strings.TrimSpace(stream.UpdatedByUserID)
	}
	result, err := s.database().Collection(collectionFormataStream).UpdateOne(
		ctx,
		bson.M{"_id": stream.ID},
		bson.M{"$set": bson.M{
			"stream":          stream.Stream,
			"updatedAt":       stream.UpdatedAt,
			"createdByUserId": stream.CreatedByUserID,
			"updatedByUserId": stream.UpdatedByUserID,
		}},
	)
	if err != nil {
		return FormataBuilderStream{}, err
	}
	if result != nil && result.MatchedCount == 0 {
		return FormataBuilderStream{}, mongo.ErrNoDocuments
	}
	return stream, nil
}

func (s *MongoStore) LoadFormataBuilderStream(ctx context.Context) (*FormataBuilderStream, error) {
	var stream FormataBuilderStream
	if err := s.database().Collection(collectionFormataStream).FindOne(
		ctx,
		bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "_id", Value: -1}}),
	).Decode(&stream); err != nil {
		return nil, err
	}
	return &stream, nil
}

func (s *MongoStore) LoadFormataBuilderStreamByID(ctx context.Context, id primitive.ObjectID) (*FormataBuilderStream, error) {
	var stream FormataBuilderStream
	if err := s.database().Collection(collectionFormataStream).FindOne(
		ctx,
		bson.M{"_id": id},
	).Decode(&stream); err != nil {
		return nil, err
	}
	return &stream, nil
}

func (s *MongoStore) ListFormataBuilderStreams(ctx context.Context) ([]FormataBuilderStream, error) {
	cursor, err := s.database().Collection(collectionFormataStream).Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []FormataBuilderStream{}
	for cursor.Next(ctx) {
		var stream FormataBuilderStream
		if err := cursor.Decode(&stream); err != nil {
			continue
		}
		items = append(items, stream)
	}
	return items, nil
}

func (s *MongoStore) DeleteFormataBuilderStream(ctx context.Context, id primitive.ObjectID) error {
	result, err := s.database().Collection(collectionFormataStream).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result != nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) DeleteWorkflowData(ctx context.Context, workflowKey string) error {
	processCursor, err := s.database().Collection("processes").Find(
		ctx,
		bson.M{"workflowKey": strings.TrimSpace(workflowKey)},
		options.Find().SetProjection(bson.M{"_id": 1}),
	)
	if err != nil {
		return err
	}
	defer processCursor.Close(ctx)

	processIDs := make([]primitive.ObjectID, 0)
	for processCursor.Next(ctx) {
		var doc bson.M
		if err := processCursor.Decode(&doc); err != nil {
			continue
		}
		id, ok := doc["_id"].(primitive.ObjectID)
		if !ok || id.IsZero() {
			continue
		}
		processIDs = append(processIDs, id)
	}
	if len(processIDs) == 0 {
		return nil
	}

	attachmentCursor, err := s.database().Collection("attachments.files").Find(
		ctx,
		bson.M{"metadata.processId": bson.M{"$in": processIDs}},
		options.Find().SetProjection(bson.M{"_id": 1}),
	)
	if err != nil {
		return err
	}
	defer attachmentCursor.Close(ctx)

	attachmentIDs := make([]primitive.ObjectID, 0)
	for attachmentCursor.Next(ctx) {
		var doc bson.M
		if err := attachmentCursor.Decode(&doc); err != nil {
			continue
		}
		id, ok := doc["_id"].(primitive.ObjectID)
		if !ok || id.IsZero() {
			continue
		}
		attachmentIDs = append(attachmentIDs, id)
	}

	if len(attachmentIDs) > 0 {
		if _, err := s.database().Collection("attachments.chunks").DeleteMany(ctx, bson.M{"files_id": bson.M{"$in": attachmentIDs}}); err != nil {
			return err
		}
		if _, err := s.database().Collection("attachments.files").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": attachmentIDs}}); err != nil {
			return err
		}
	}

	if _, err := s.database().Collection("notarizations").DeleteMany(ctx, bson.M{"processId": bson.M{"$in": processIDs}}); err != nil {
		return err
	}
	if _, err := s.database().Collection("processes").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": processIDs}}); err != nil {
		return err
	}
	return nil
}
