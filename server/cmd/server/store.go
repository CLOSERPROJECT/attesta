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
	UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, workflowKey, substepID string, progress ProcessStep) error
	UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, workflowKey, status string) error
	UpdateProcessDPP(ctx context.Context, id primitive.ObjectID, workflowKey string, dpp ProcessDPP) error
	InsertNotarization(ctx context.Context, notarization Notarization) error
	SaveAttachment(ctx context.Context, upload AttachmentUpload, content io.Reader) (Attachment, error)
	LoadAttachmentByID(ctx context.Context, id primitive.ObjectID) (*Attachment, error)
	OpenAttachmentDownload(ctx context.Context, id primitive.ObjectID) (io.ReadCloser, error)
	CreateOrganization(ctx context.Context, org Organization) (Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	ListOrganizations(ctx context.Context) ([]Organization, error)
	CreateRole(ctx context.Context, role Role) (Role, error)
	GetRoleBySlug(ctx context.Context, orgSlug, roleSlug string) (*Role, error)
	ListRolesByOrg(ctx context.Context, orgSlug string) ([]Role, error)
	CreateUser(ctx context.Context, user AccountUser) (AccountUser, error)
	GetUserByEmail(ctx context.Context, email string) (*AccountUser, error)
	GetUserByUserID(ctx context.Context, userID string) (*AccountUser, error)
	SetUserPasswordHash(ctx context.Context, userID, passwordHash string) error
	SetUserRoles(ctx context.Context, userID string, roleSlugs []string) error
	SetUserLastLogin(ctx context.Context, userID string, lastLoginAt time.Time) error
	CreateInvite(ctx context.Context, invite Invite) (Invite, error)
	LoadInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error)
	MarkInviteUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
	CreateSession(ctx context.Context, session Session) (Session, error)
	LoadSessionByID(ctx context.Context, sessionID string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	CreatePasswordReset(ctx context.Context, reset PasswordReset) (PasswordReset, error)
	LoadPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordReset, error)
	MarkPasswordResetUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
}

type Organization struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Slug      string             `bson:"slug"`
	Name      string             `bson:"name"`
	Color     string             `bson:"color,omitempty"`
	Border    string             `bson:"border,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
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
	UserID          string              `bson:"userId"`
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

type Invite struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	OrgID           primitive.ObjectID `bson:"orgId"`
	Email           string             `bson:"email"`
	UserID          string             `bson:"userId"`
	RoleSlugs       []string           `bson:"roleSlugs"`
	TokenHash       string             `bson:"tokenHash"`
	ExpiresAt       time.Time          `bson:"expiresAt"`
	UsedAt          *time.Time         `bson:"usedAt,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt"`
	CreatedByUserID string             `bson:"createdByUserId"`
}

type Session struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty"`
	SessionID   string              `bson:"sessionId"`
	UserID      string              `bson:"userId"`
	UserMongoID primitive.ObjectID  `bson:"userMongoId"`
	OrgID       *primitive.ObjectID `bson:"orgId,omitempty"`
	CreatedAt   time.Time           `bson:"createdAt"`
	LastLoginAt time.Time           `bson:"lastLoginAt"`
	ExpiresAt   time.Time           `bson:"expiresAt"`
}

type PasswordReset struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Email     string             `bson:"email"`
	UserID    string             `bson:"userId"`
	TokenHash string             `bson:"tokenHash"`
	ExpiresAt time.Time          `bson:"expiresAt"`
	UsedAt    *time.Time         `bson:"usedAt,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
}

const (
	collectionOrganizations = "organizations"
	collectionRoles         = "roles"
	collectionUsers         = "users"
	collectionInvites       = "invites"
	collectionSessions      = "sessions"
	collectionPasswordReset = "password_resets"
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
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort
	CreateIndexes(ctx context.Context, models []mongo.IndexModel) error
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

func (c mongoDriverCollection) CreateIndexes(ctx context.Context, models []mongo.IndexModel) error {
	_, err := c.collection.Indexes().CreateMany(ctx, models)
	return err
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
	mu            sync.RWMutex
	processes     map[primitive.ObjectID]Process
	notarizations []Notarization
	attachments   map[primitive.ObjectID]memoryAttachment
	organizations map[string]Organization
	roles         map[string]Role
	usersByUserID map[string]AccountUser
	usersByEmail  map[string]string
	invites       map[string]Invite
	sessions      map[string]Session
	resets        map[string]PasswordReset

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
		processes:     map[primitive.ObjectID]Process{},
		attachments:   map[primitive.ObjectID]memoryAttachment{},
		organizations: map[string]Organization{},
		roles:         map[string]Role{},
		usersByUserID: map[string]AccountUser{},
		usersByEmail:  map[string]string{},
		invites:       map[string]Invite{},
		sessions:      map[string]Session{},
		resets:        map[string]PasswordReset{},
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

func (s *MemoryStore) CreateOrganization(_ context.Context, org Organization) (Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if org.CreatedAt.IsZero() {
		org.CreatedAt = now
	}
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		org.Name = strings.TrimSpace(org.Slug)
	}
	org.Slug = canonifySlug(org.Name)
	if org.Slug == "" {
		return Organization{}, errors.New("organization slug required")
	}
	if _, exists := s.organizations[org.Slug]; exists {
		return Organization{}, errors.New("organization slug already exists")
	}
	if org.ID.IsZero() {
		org.ID = primitive.NewObjectID()
	}
	s.organizations[org.Slug] = org
	return org, nil
}

func (s *MemoryStore) GetOrganizationBySlug(_ context.Context, slug string) (*Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, ok := s.organizations[canonifySlug(slug)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := org
	return &copy, nil
}

func (s *MemoryStore) ListOrganizations(_ context.Context) ([]Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	organizations := make([]Organization, 0, len(s.organizations))
	for _, org := range s.organizations {
		organizations = append(organizations, org)
	}
	sort.Slice(organizations, func(i, j int) bool {
		return organizations[i].Name < organizations[j].Name
	})
	return organizations, nil
}

func (s *MemoryStore) CreateRole(_ context.Context, role Role) (Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role.OrgSlug = canonifySlug(role.OrgSlug)
	if _, exists := s.organizations[role.OrgSlug]; !exists {
		return Role{}, errors.New("organization not found")
	}
	role.Name = strings.TrimSpace(role.Name)
	if role.Name == "" {
		role.Name = strings.TrimSpace(role.Slug)
	}
	role.Slug = canonifySlug(role.Name)
	if role.Slug == "" {
		return Role{}, errors.New("role slug required")
	}
	if role.CreatedAt.IsZero() {
		role.CreatedAt = time.Now().UTC()
	}
	key := role.OrgSlug + ":" + role.Slug
	if _, exists := s.roles[key]; exists {
		return Role{}, errors.New("role already exists")
	}
	if role.ID.IsZero() {
		role.ID = primitive.NewObjectID()
	}
	if role.OrgID.IsZero() {
		org := s.organizations[role.OrgSlug]
		role.OrgID = org.ID
	}
	s.roles[key] = role
	return role, nil
}

func (s *MemoryStore) GetRoleBySlug(_ context.Context, orgSlug, roleSlug string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := canonifySlug(orgSlug) + ":" + canonifySlug(roleSlug)
	role, ok := s.roles[key]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := role
	return &copy, nil
}

func (s *MemoryStore) ListRolesByOrg(_ context.Context, orgSlug string) ([]Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canonOrg := canonifySlug(orgSlug)
	roles := []Role{}
	for key, role := range s.roles {
		if strings.HasPrefix(key, canonOrg+":") {
			roles = append(roles, role)
		}
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})
	return roles, nil
}

func (s *MemoryStore) CreateUser(_ context.Context, user AccountUser) (AccountUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.UserID = strings.TrimSpace(user.UserID)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.OrgSlug = canonifySlug(user.OrgSlug)
	user.RoleSlugs = canonifyRoleSlugs(user.RoleSlugs)
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	if user.UserID == "" || user.Email == "" {
		return AccountUser{}, errors.New("user id and email required")
	}
	if _, exists := s.usersByUserID[user.UserID]; exists {
		return AccountUser{}, errors.New("user id already exists")
	}
	if _, exists := s.usersByEmail[user.Email]; exists {
		return AccountUser{}, errors.New("email already exists")
	}
	if user.OrgSlug != "" {
		org, exists := s.organizations[user.OrgSlug]
		if !exists {
			return AccountUser{}, errors.New("organization not found")
		}
		if user.OrgID == nil {
			orgID := org.ID
			user.OrgID = &orgID
		}
		for _, roleSlug := range user.RoleSlugs {
			if _, exists := s.roles[user.OrgSlug+":"+roleSlug]; !exists {
				return AccountUser{}, errors.New("role not found in organization")
			}
		}
	}
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	s.usersByUserID[user.UserID] = user
	s.usersByEmail[user.Email] = user.UserID
	return user, nil
}

func (s *MemoryStore) GetUserByEmail(_ context.Context, email string) (*AccountUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	user := s.usersByUserID[userID]
	copy := user
	return &copy, nil
}

func (s *MemoryStore) GetUserByUserID(_ context.Context, userID string) (*AccountUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.usersByUserID[strings.TrimSpace(userID)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := user
	return &copy, nil
}

func (s *MemoryStore) SetUserPasswordHash(_ context.Context, userID, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.usersByUserID[strings.TrimSpace(userID)]
	if !ok {
		return mongo.ErrNoDocuments
	}
	user.PasswordHash = strings.TrimSpace(passwordHash)
	s.usersByUserID[user.UserID] = user
	return nil
}

func (s *MemoryStore) SetUserRoles(_ context.Context, userID string, roleSlugs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.usersByUserID[strings.TrimSpace(userID)]
	if !ok {
		return mongo.ErrNoDocuments
	}
	canonRoles := canonifyRoleSlugs(roleSlugs)
	for _, roleSlug := range canonRoles {
		if _, exists := s.roles[user.OrgSlug+":"+roleSlug]; !exists {
			return errors.New("role not found in organization")
		}
	}
	user.RoleSlugs = canonRoles
	s.usersByUserID[user.UserID] = user
	return nil
}

func (s *MemoryStore) SetUserLastLogin(_ context.Context, userID string, lastLoginAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.usersByUserID[strings.TrimSpace(userID)]
	if !ok {
		return mongo.ErrNoDocuments
	}
	timestamp := lastLoginAt.UTC()
	user.LastLoginAt = &timestamp
	s.usersByUserID[user.UserID] = user
	return nil
}

func (s *MemoryStore) CreateInvite(_ context.Context, invite Invite) (Invite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	invite.Email = strings.ToLower(strings.TrimSpace(invite.Email))
	invite.TokenHash = hashLookupToken(invite.TokenHash)
	invite.RoleSlugs = canonifyRoleSlugs(invite.RoleSlugs)
	if invite.CreatedAt.IsZero() {
		invite.CreatedAt = time.Now().UTC()
	}
	if _, exists := s.invites[invite.TokenHash]; exists {
		return Invite{}, errors.New("invite token already exists")
	}
	if invite.ID.IsZero() {
		invite.ID = primitive.NewObjectID()
	}
	s.invites[invite.TokenHash] = invite
	return invite, nil
}

func (s *MemoryStore) LoadInviteByTokenHash(_ context.Context, tokenHash string) (*Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	invite, ok := s.invites[hashLookupToken(tokenHash)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := invite
	return &copy, nil
}

func (s *MemoryStore) MarkInviteUsed(_ context.Context, tokenHash string, usedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := hashLookupToken(tokenHash)
	invite, ok := s.invites[hash]
	if !ok {
		return mongo.ErrNoDocuments
	}
	if invite.UsedAt != nil {
		return errors.New("invite already used")
	}
	timestamp := usedAt.UTC()
	invite.UsedAt = &timestamp
	s.invites[hash] = invite
	return nil
}

func (s *MemoryStore) CreateSession(_ context.Context, session Session) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.SessionID = strings.TrimSpace(session.SessionID)
	if session.SessionID == "" {
		return Session{}, errors.New("session id required")
	}
	if _, exists := s.sessions[session.SessionID]; exists {
		return Session{}, errors.New("session id already exists")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.ID.IsZero() {
		session.ID = primitive.NewObjectID()
	}
	s.sessions[session.SessionID] = session
	return session, nil
}

func (s *MemoryStore) LoadSessionByID(_ context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[strings.TrimSpace(sessionID)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := session
	return &copy, nil
}

func (s *MemoryStore) DeleteSession(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(sessionID)
	if _, ok := s.sessions[key]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.sessions, key)
	return nil
}

func (s *MemoryStore) CreatePasswordReset(_ context.Context, reset PasswordReset) (PasswordReset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reset.Email = strings.ToLower(strings.TrimSpace(reset.Email))
	reset.TokenHash = hashLookupToken(reset.TokenHash)
	if reset.CreatedAt.IsZero() {
		reset.CreatedAt = time.Now().UTC()
	}
	if _, exists := s.resets[reset.TokenHash]; exists {
		return PasswordReset{}, errors.New("reset token already exists")
	}
	if reset.ID.IsZero() {
		reset.ID = primitive.NewObjectID()
	}
	s.resets[reset.TokenHash] = reset
	return reset, nil
}

func (s *MemoryStore) LoadPasswordResetByTokenHash(_ context.Context, tokenHash string) (*PasswordReset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reset, ok := s.resets[hashLookupToken(tokenHash)]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := reset
	return &copy, nil
}

func (s *MemoryStore) MarkPasswordResetUsed(_ context.Context, tokenHash string, usedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := hashLookupToken(tokenHash)
	reset, ok := s.resets[hash]
	if !ok {
		return mongo.ErrNoDocuments
	}
	if reset.UsedAt != nil {
		return errors.New("password reset already used")
	}
	timestamp := usedAt.UTC()
	reset.UsedAt = &timestamp
	s.resets[hash] = reset
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

func hashLookupToken(token string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(hash[:])
}

func (s *MongoStore) EnsureAuthIndexes(ctx context.Context) error {
	indexes := map[string][]mongo.IndexModel{
		collectionOrganizations: {
			{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		collectionRoles: {
			{Keys: bson.D{{Key: "orgId", Value: 1}, {Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		collectionUsers: {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "orgId", Value: 1}}},
		},
		collectionInvites: {
			{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			{Keys: bson.D{{Key: "email", Value: 1}}},
		},
		collectionSessions: {
			{Keys: bson.D{{Key: "sessionId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			{Keys: bson.D{{Key: "userId", Value: 1}}},
		},
		collectionPasswordReset: {
			{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
	}

	for name, models := range indexes {
		if err := s.database().Collection(name).CreateIndexes(ctx, models); err != nil {
			return err
		}
	}
	return nil
}

func (s *MongoStore) CreateOrganization(ctx context.Context, org Organization) (Organization, error) {
	now := time.Now().UTC()
	if org.CreatedAt.IsZero() {
		org.CreatedAt = now
	}
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		org.Name = strings.TrimSpace(org.Slug)
	}
	org.Slug = canonifySlug(org.Name)
	result, err := s.database().Collection(collectionOrganizations).InsertOne(ctx, org)
	if err != nil {
		return Organization{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		org.ID = insertedID
	}
	return org, nil
}

func (s *MongoStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	if err := s.database().Collection(collectionOrganizations).FindOne(ctx, bson.M{"slug": canonifySlug(slug)}).Decode(&org); err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *MongoStore) ListOrganizations(ctx context.Context) ([]Organization, error) {
	cursor, err := s.database().Collection(collectionOrganizations).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	organizations := []Organization{}
	for cursor.Next(ctx) {
		var org Organization
		if err := cursor.Decode(&org); err != nil {
			continue
		}
		organizations = append(organizations, org)
	}
	return organizations, nil
}

func (s *MongoStore) CreateRole(ctx context.Context, role Role) (Role, error) {
	if role.CreatedAt.IsZero() {
		role.CreatedAt = time.Now().UTC()
	}
	role.Name = strings.TrimSpace(role.Name)
	if role.Name == "" {
		role.Name = strings.TrimSpace(role.Slug)
	}
	role.Slug = canonifySlug(role.Name)
	role.OrgSlug = canonifySlug(role.OrgSlug)
	result, err := s.database().Collection(collectionRoles).InsertOne(ctx, role)
	if err != nil {
		return Role{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		role.ID = insertedID
	}
	return role, nil
}

func (s *MongoStore) GetRoleBySlug(ctx context.Context, orgSlug, roleSlug string) (*Role, error) {
	var role Role
	filter := bson.M{
		"orgSlug": canonifySlug(orgSlug),
		"slug":    canonifySlug(roleSlug),
	}
	if err := s.database().Collection(collectionRoles).FindOne(ctx, filter).Decode(&role); err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *MongoStore) ListRolesByOrg(ctx context.Context, orgSlug string) ([]Role, error) {
	filter := bson.M{"orgSlug": canonifySlug(orgSlug)}
	cursor, err := s.database().Collection(collectionRoles).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	roles := []Role{}
	for cursor.Next(ctx) {
		var role Role
		if err := cursor.Decode(&role); err != nil {
			continue
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (s *MongoStore) CreateUser(ctx context.Context, user AccountUser) (AccountUser, error) {
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.RoleSlugs = canonifyRoleSlugs(user.RoleSlugs)
	user.OrgSlug = canonifySlug(user.OrgSlug)
	result, err := s.database().Collection(collectionUsers).InsertOne(ctx, user)
	if err != nil {
		return AccountUser{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = insertedID
	}
	return user, nil
}

func (s *MongoStore) GetUserByEmail(ctx context.Context, email string) (*AccountUser, error) {
	var user AccountUser
	if err := s.database().Collection(collectionUsers).FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *MongoStore) GetUserByUserID(ctx context.Context, userID string) (*AccountUser, error) {
	var user AccountUser
	if err := s.database().Collection(collectionUsers).FindOne(ctx, bson.M{"userId": strings.TrimSpace(userID)}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *MongoStore) SetUserPasswordHash(ctx context.Context, userID, passwordHash string) error {
	_, err := s.database().Collection(collectionUsers).UpdateOne(
		ctx,
		bson.M{"userId": strings.TrimSpace(userID)},
		bson.M{"$set": bson.M{"passwordHash": strings.TrimSpace(passwordHash)}},
	)
	return err
}

func (s *MongoStore) SetUserRoles(ctx context.Context, userID string, roleSlugs []string) error {
	_, err := s.database().Collection(collectionUsers).UpdateOne(
		ctx,
		bson.M{"userId": strings.TrimSpace(userID)},
		bson.M{"$set": bson.M{"roleSlugs": canonifyRoleSlugs(roleSlugs)}},
	)
	return err
}

func (s *MongoStore) SetUserLastLogin(ctx context.Context, userID string, lastLoginAt time.Time) error {
	_, err := s.database().Collection(collectionUsers).UpdateOne(
		ctx,
		bson.M{"userId": strings.TrimSpace(userID)},
		bson.M{"$set": bson.M{"lastLoginAt": lastLoginAt.UTC()}},
	)
	return err
}

func (s *MongoStore) CreateInvite(ctx context.Context, invite Invite) (Invite, error) {
	if invite.CreatedAt.IsZero() {
		invite.CreatedAt = time.Now().UTC()
	}
	invite.Email = strings.ToLower(strings.TrimSpace(invite.Email))
	invite.TokenHash = hashLookupToken(invite.TokenHash)
	invite.RoleSlugs = canonifyRoleSlugs(invite.RoleSlugs)
	result, err := s.database().Collection(collectionInvites).InsertOne(ctx, invite)
	if err != nil {
		return Invite{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		invite.ID = insertedID
	}
	return invite, nil
}

func (s *MongoStore) LoadInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error) {
	var invite Invite
	if err := s.database().Collection(collectionInvites).FindOne(ctx, bson.M{"tokenHash": hashLookupToken(tokenHash)}).Decode(&invite); err != nil {
		return nil, err
	}
	return &invite, nil
}

func (s *MongoStore) MarkInviteUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	_, err := s.database().Collection(collectionInvites).UpdateOne(
		ctx,
		bson.M{"tokenHash": hashLookupToken(tokenHash), "usedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"usedAt": usedAt.UTC()}},
	)
	return err
}

func (s *MongoStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	session.SessionID = strings.TrimSpace(session.SessionID)
	result, err := s.database().Collection(collectionSessions).InsertOne(ctx, session)
	if err != nil {
		return Session{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		session.ID = insertedID
	}
	return session, nil
}

func (s *MongoStore) LoadSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	if err := s.database().Collection(collectionSessions).FindOne(ctx, bson.M{"sessionId": strings.TrimSpace(sessionID)}).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *MongoStore) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.database().Collection(collectionSessions).UpdateOne(
		ctx,
		bson.M{"sessionId": strings.TrimSpace(sessionID)},
		bson.M{"$set": bson.M{"expiresAt": time.Now().UTC()}},
	)
	return err
}

func (s *MongoStore) CreatePasswordReset(ctx context.Context, reset PasswordReset) (PasswordReset, error) {
	if reset.CreatedAt.IsZero() {
		reset.CreatedAt = time.Now().UTC()
	}
	reset.Email = strings.ToLower(strings.TrimSpace(reset.Email))
	reset.TokenHash = hashLookupToken(reset.TokenHash)
	result, err := s.database().Collection(collectionPasswordReset).InsertOne(ctx, reset)
	if err != nil {
		return PasswordReset{}, err
	}
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		reset.ID = insertedID
	}
	return reset, nil
}

func (s *MongoStore) LoadPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordReset, error) {
	var reset PasswordReset
	if err := s.database().Collection(collectionPasswordReset).FindOne(ctx, bson.M{"tokenHash": hashLookupToken(tokenHash)}).Decode(&reset); err != nil {
		return nil, err
	}
	return &reset, nil
}

func (s *MongoStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	_, err := s.database().Collection(collectionPasswordReset).UpdateOne(
		ctx,
		bson.M{"tokenHash": hashLookupToken(tokenHash), "usedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"usedAt": usedAt.UTC()}},
	)
	return err
}
