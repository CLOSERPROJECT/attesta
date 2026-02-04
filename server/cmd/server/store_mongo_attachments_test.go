package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoStoreSaveAttachmentDefaultsAndMetadataUpdate(t *testing.T) {
	bucket := &fakeGridFSBucket{}
	filesCollection := &fakeMongoCollection{}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{"attachments.files": filesCollection},
		bucket:      bucket,
	}
	store := &MongoStore{dbPort: db}

	processID := primitive.NewObjectID()
	content := []byte("hello attachment")
	before := time.Now().UTC()
	attachment, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID: processID,
		SubstepID: "1.3",
		MaxBytes:  1024,
	}, bytes.NewReader(content))
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("SaveAttachment returned error: %v", err)
	}

	if attachment.Filename != "attachment" {
		t.Fatalf("filename = %q, want attachment", attachment.Filename)
	}
	if attachment.ContentType != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", attachment.ContentType)
	}
	if attachment.UploadedAt.Before(before) || attachment.UploadedAt.After(after.Add(2*time.Second)) {
		t.Fatalf("uploadedAt %s out of expected range [%s, %s]", attachment.UploadedAt, before, after)
	}
	if attachment.SizeBytes != int64(len(content)) {
		t.Fatalf("size = %d, want %d", attachment.SizeBytes, len(content))
	}

	sum := sha256.Sum256(content)
	wantSHA := hex.EncodeToString(sum[:])
	if attachment.SHA256 != wantSHA {
		t.Fatalf("sha256 = %q, want %q", attachment.SHA256, wantSHA)
	}
	if len(bucket.uploadedNames) != 1 || bucket.uploadedNames[0] != "attachment" {
		t.Fatalf("uploaded filenames = %#v, want [attachment]", bucket.uploadedNames)
	}
	if len(bucket.uploadOptions) != 1 || len(bucket.uploadOptions[0]) != 1 {
		t.Fatalf("upload options = %#v, want one option set", bucket.uploadOptions)
	}
	metadata, ok := bucket.uploadOptions[0][0].Metadata.(bson.M)
	if !ok {
		t.Fatalf("upload metadata type = %T, want bson.M", bucket.uploadOptions[0][0].Metadata)
	}
	if metadata["processId"] != processID || metadata["substepId"] != "1.3" || metadata["contentType"] != "application/octet-stream" {
		t.Fatalf("unexpected upload metadata: %#v", metadata)
	}
	if _, ok := metadata["uploadedAt"].(time.Time); !ok {
		t.Fatalf("expected uploadedAt time in metadata, got %#v", metadata["uploadedAt"])
	}

	if len(filesCollection.updateOneFilters) != 1 || len(filesCollection.updateOneUpdates) != 1 {
		t.Fatalf("expected metadata sha update call, got filters=%d updates=%d", len(filesCollection.updateOneFilters), len(filesCollection.updateOneUpdates))
	}
	expectedFilter := bson.M{"_id": attachment.ID}
	if !reflect.DeepEqual(filesCollection.updateOneFilters[0], expectedFilter) {
		t.Fatalf("sha update filter = %#v, want %#v", filesCollection.updateOneFilters[0], expectedFilter)
	}
	expectedUpdate := bson.M{"$set": bson.M{"metadata.sha256": wantSHA}}
	if !reflect.DeepEqual(filesCollection.updateOneUpdates[0], expectedUpdate) {
		t.Fatalf("sha update doc = %#v, want %#v", filesCollection.updateOneUpdates[0], expectedUpdate)
	}
}

func TestMongoStoreSaveAttachmentTooLargeDeletesUpload(t *testing.T) {
	bucket := &fakeGridFSBucket{}
	filesCollection := &fakeMongoCollection{}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{"attachments.files": filesCollection},
		bucket:      bucket,
	}
	store := &MongoStore{dbPort: db}

	_, err := store.SaveAttachment(t.Context(), AttachmentUpload{
		ProcessID: primitive.NewObjectID(),
		SubstepID: "1.3",
		Filename:  "file.txt",
		MaxBytes:  4,
	}, strings.NewReader("toolarge"))
	if !errors.Is(err, ErrAttachmentTooLarge) {
		t.Fatalf("SaveAttachment error = %v, want ErrAttachmentTooLarge", err)
	}
	if len(bucket.deletedIDs) != 1 {
		t.Fatalf("expected one bucket delete call, got %d", len(bucket.deletedIDs))
	}
	if len(filesCollection.updateOneUpdates) != 0 {
		t.Fatalf("expected no sha update after too-large error, got %#v", filesCollection.updateOneUpdates)
	}
}

func TestMongoStoreLoadAttachmentByIDUsesUploadDateFallback(t *testing.T) {
	attachmentID := primitive.NewObjectID()
	processID := primitive.NewObjectID()
	uploadDate := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	filesCollection := &fakeMongoCollection{
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				doc := reflect.ValueOf(v).Elem()
				doc.FieldByName("ID").Set(reflect.ValueOf(attachmentID))
				doc.FieldByName("Filename").SetString("cert.pdf")
				doc.FieldByName("Length").SetInt(42)
				doc.FieldByName("UploadDate").Set(reflect.ValueOf(uploadDate))

				meta := doc.FieldByName("Metadata")
				meta.FieldByName("ProcessID").Set(reflect.ValueOf(processID))
				meta.FieldByName("SubstepID").SetString("1.3")
				meta.FieldByName("ContentType").SetString("application/pdf")
				meta.FieldByName("SHA256").SetString("abc123")
				return nil
			}}
		},
	}
	db := &fakeMongoDatabase{
		collections: map[string]*fakeMongoCollection{"attachments.files": filesCollection},
		bucket:      &fakeGridFSBucket{},
	}
	store := &MongoStore{dbPort: db}

	attachment, err := store.LoadAttachmentByID(t.Context(), attachmentID)
	if err != nil {
		t.Fatalf("LoadAttachmentByID returned error: %v", err)
	}
	if attachment.UploadedAt != uploadDate {
		t.Fatalf("uploadedAt = %s, want fallback uploadDate %s", attachment.UploadedAt, uploadDate)
	}
	if attachment.ProcessID != processID || attachment.Filename != "cert.pdf" || attachment.ContentType != "application/pdf" {
		t.Fatalf("attachment metadata mismatch: %#v", attachment)
	}
}

func TestMongoStoreOpenAttachmentDownloadAndBucketName(t *testing.T) {
	attachmentID := primitive.NewObjectID()
	openErr := errors.New("open failed")
	var openedID interface{}

	bucket := &fakeGridFSBucket{
		openFn: func(fileID interface{}) (io.ReadCloser, error) {
			openedID = fileID
			return nil, openErr
		},
	}
	db := &fakeMongoDatabase{bucket: bucket}
	store := &MongoStore{dbPort: db}

	if _, err := store.OpenAttachmentDownload(t.Context(), attachmentID); !errors.Is(err, openErr) {
		t.Fatalf("OpenAttachmentDownload error = %v, want %v", err, openErr)
	}
	if openedID != attachmentID {
		t.Fatalf("open stream id = %#v, want %#v", openedID, attachmentID)
	}
	if len(db.bucketNames) != 1 || db.bucketNames[0] != "attachments" {
		t.Fatalf("bucket name calls = %#v, want [attachments]", db.bucketNames)
	}
}
