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
	LoadLatestProcess(ctx context.Context) (*Process, error)
	ListRecentProcesses(ctx context.Context, limit int64) ([]Process, error)
	UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, substepID string, progress ProcessStep) error
	UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, status string) error
	InsertNotarization(ctx context.Context, notarization Notarization) error
	SaveAttachment(ctx context.Context, upload AttachmentUpload, content io.Reader) (Attachment, error)
	LoadAttachmentByID(ctx context.Context, id primitive.ObjectID) (*Attachment, error)
	OpenAttachmentDownload(ctx context.Context, id primitive.ObjectID) (io.ReadCloser, error)
}

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
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort
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

func (c mongoDriverCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
	return mongoDriverSingleResult{result: c.collection.FindOneAndUpdate(ctx, filter, update, opts...)}
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

func (s *MongoStore) LoadLatestProcess(ctx context.Context) (*Process, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var process Process
	if err := s.database().Collection("processes").FindOne(ctx, bson.M{}, opts).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) ListRecentProcesses(ctx context.Context, limit int64) ([]Process, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cursor, err := s.database().Collection("processes").Find(ctx, bson.M{}, opts)
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

func (s *MongoStore) UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, substepID string, progress ProcessStep) error {
	update := bson.M{
		"$set": bson.M{
			"progress." + encodeProgressKey(substepID): progress,
		},
	}
	return s.database().Collection("processes").FindOneAndUpdate(ctx, bson.M{"_id": id}, update).Err()
}

func (s *MongoStore) UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	_, err := s.database().Collection("processes").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status}})
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
	mu            sync.RWMutex
	processes     map[primitive.ObjectID]Process
	notarizations []Notarization
	attachments   map[primitive.ObjectID]memoryAttachment

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
		processes:   map[primitive.ObjectID]Process{},
		attachments: map[primitive.ObjectID]memoryAttachment{},
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

func (s *MemoryStore) LoadLatestProcess(_ context.Context) (*Process, error) {
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
		if first || process.CreatedAt.After(latest.CreatedAt) {
			latest = process
			first = false
		}
	}
	cloned := cloneProcess(latest)
	return &cloned, nil
}

func (s *MemoryStore) ListRecentProcesses(_ context.Context, limit int64) ([]Process, error) {
	if s.ListProcessesErr != nil {
		return nil, s.ListProcessesErr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Process, 0, len(s.processes))
	for _, process := range s.processes {
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

func (s *MemoryStore) UpdateProcessProgress(_ context.Context, id primitive.ObjectID, substepID string, progress ProcessStep) error {
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
	process.Progress[encodeProgressKey(substepID)] = cloneProcessStep(progress)
	s.processes[id] = process
	return nil
}

func (s *MemoryStore) UpdateProcessStatus(_ context.Context, id primitive.ObjectID, status string) error {
	if s.UpdateStatusErr != nil {
		return s.UpdateStatusErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	process, ok := s.processes[id]
	if !ok {
		return mongo.ErrNoDocuments
	}
	process.Status = status
	s.processes[id] = process
	return nil
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

func cloneProcess(process Process) Process {
	cloned := process
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
