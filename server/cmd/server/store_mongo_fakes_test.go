package main

import (
	"context"
	"errors"
	"io"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakeMongoDatabase struct {
	collections map[string]*fakeMongoCollection
	bucket      gridFSBucketPort
	bucketErr   error
	bucketNames []string
}

func (db *fakeMongoDatabase) Collection(name string) mongoCollectionPort {
	if db.collections == nil {
		db.collections = map[string]*fakeMongoCollection{}
	}
	if collection, ok := db.collections[name]; ok {
		return collection
	}
	collection := &fakeMongoCollection{}
	db.collections[name] = collection
	return collection
}

func (db *fakeMongoDatabase) NewGridFSBucket(name string) (gridFSBucketPort, error) {
	db.bucketNames = append(db.bucketNames, name)
	if db.bucketErr != nil {
		return nil, db.bucketErr
	}
	if db.bucket == nil {
		return nil, errors.New("missing fake bucket")
	}
	return db.bucket, nil
}

type fakeMongoCollection struct {
	insertOneFn         func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	findOneFn           func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort
	findFn              func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error)
	updateOneFn         func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	findOneAndUpdateFn  func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort
	insertDocuments     []interface{}
	findOneFilters      []interface{}
	findOneOptionsCalls [][]*options.FindOneOptions
	findFilters         []interface{}
	findOptionsCalls    [][]*options.FindOptions
	updateOneFilters    []interface{}
	updateOneUpdates    []interface{}
	findOneAndUpdFilter []interface{}
	findOneAndUpdUpdate []interface{}
	createIndexesFn     func(ctx context.Context, models []mongo.IndexModel) error
	createIndexesModels [][]mongo.IndexModel
}

func (c *fakeMongoCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	c.insertDocuments = append(c.insertDocuments, document)
	if c.insertOneFn != nil {
		return c.insertOneFn(ctx, document, opts...)
	}
	return &mongo.InsertOneResult{}, nil
}

func (c *fakeMongoCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
	c.findOneFilters = append(c.findOneFilters, filter)
	c.findOneOptionsCalls = append(c.findOneOptionsCalls, opts)
	if c.findOneFn != nil {
		return c.findOneFn(ctx, filter, opts...)
	}
	return fakeSingleResult{err: mongo.ErrNoDocuments}
}

func (c *fakeMongoCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
	c.findFilters = append(c.findFilters, filter)
	c.findOptionsCalls = append(c.findOptionsCalls, opts)
	if c.findFn != nil {
		return c.findFn(ctx, filter, opts...)
	}
	return nil, errors.New("find not configured")
}

func (c *fakeMongoCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	c.updateOneFilters = append(c.updateOneFilters, filter)
	c.updateOneUpdates = append(c.updateOneUpdates, update)
	if c.updateOneFn != nil {
		return c.updateOneFn(ctx, filter, update, opts...)
	}
	return &mongo.UpdateResult{}, nil
}

func (c *fakeMongoCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) mongoSingleResultPort {
	c.findOneAndUpdFilter = append(c.findOneAndUpdFilter, filter)
	c.findOneAndUpdUpdate = append(c.findOneAndUpdUpdate, update)
	if c.findOneAndUpdateFn != nil {
		return c.findOneAndUpdateFn(ctx, filter, update, opts...)
	}
	return fakeSingleResult{}
}

func (c *fakeMongoCollection) CreateIndexes(ctx context.Context, models []mongo.IndexModel) error {
	c.createIndexesModels = append(c.createIndexesModels, models)
	if c.createIndexesFn != nil {
		return c.createIndexesFn(ctx, models)
	}
	return nil
}

type fakeSingleResult struct {
	decodeFn func(v interface{}) error
	err      error
}

func (r fakeSingleResult) Decode(v interface{}) error {
	if r.decodeFn != nil {
		return r.decodeFn(v)
	}
	return r.err
}

func (r fakeSingleResult) Err() error {
	return r.err
}

type fakeCursor struct {
	docs        []Process
	decodeErrAt map[int]error
	index       int
	closed      bool
	closeErr    error
}

func (c *fakeCursor) Next(ctx context.Context) bool {
	return c.index < len(c.docs)
}

func (c *fakeCursor) Decode(val interface{}) error {
	if err, ok := c.decodeErrAt[c.index]; ok {
		c.index++
		return err
	}
	target, ok := val.(*Process)
	if !ok {
		return errors.New("unsupported decode target")
	}
	*target = c.docs[c.index]
	c.index++
	return nil
}

func (c *fakeCursor) Close(ctx context.Context) error {
	c.closed = true
	return c.closeErr
}

type fakeAnyCursor struct {
	items       []interface{}
	decodeErrAt map[int]error
	index       int
	closed      bool
	closeErr    error
}

func (c *fakeAnyCursor) Next(ctx context.Context) bool {
	return c.index < len(c.items)
}

func (c *fakeAnyCursor) Decode(val interface{}) error {
	if err, ok := c.decodeErrAt[c.index]; ok {
		c.index++
		return err
	}
	item := c.items[c.index]
	c.index++
	switch target := val.(type) {
	case *Organization:
		if v, ok := item.(Organization); ok {
			*target = v
			return nil
		}
	case *Role:
		if v, ok := item.(Role); ok {
			*target = v
			return nil
		}
	case *AccountUser:
		if v, ok := item.(AccountUser); ok {
			*target = v
			return nil
		}
	case *Session:
		if v, ok := item.(Session); ok {
			*target = v
			return nil
		}
	case *Invite:
		if v, ok := item.(Invite); ok {
			*target = v
			return nil
		}
	case *PasswordReset:
		if v, ok := item.(PasswordReset); ok {
			*target = v
			return nil
		}
	case *Process:
		if v, ok := item.(Process); ok {
			*target = v
			return nil
		}
	}
	return errors.New("unsupported decode target")
}

func (c *fakeAnyCursor) Close(ctx context.Context) error {
	c.closed = true
	return c.closeErr
}

type fakeGridFSBucket struct {
	uploadFn      func(id interface{}, filename string, source io.Reader, opts ...*options.UploadOptions) error
	openFn        func(fileID interface{}) (io.ReadCloser, error)
	deleteFn      func(fileID interface{}) error
	uploadedIDs   []interface{}
	uploadedNames []string
	uploadOptions [][]*options.UploadOptions
	deletedIDs    []interface{}
}

func (b *fakeGridFSBucket) UploadFromStreamWithID(id interface{}, filename string, source io.Reader, opts ...*options.UploadOptions) error {
	b.uploadedIDs = append(b.uploadedIDs, id)
	b.uploadedNames = append(b.uploadedNames, filename)
	b.uploadOptions = append(b.uploadOptions, opts)
	if b.uploadFn != nil {
		return b.uploadFn(id, filename, source, opts...)
	}
	_, err := io.Copy(io.Discard, source)
	return err
}

func (b *fakeGridFSBucket) OpenDownloadStream(fileID interface{}) (io.ReadCloser, error) {
	if b.openFn != nil {
		return b.openFn(fileID)
	}
	return nil, errors.New("download stream not configured")
}

func (b *fakeGridFSBucket) Delete(fileID interface{}) error {
	b.deletedIDs = append(b.deletedIDs, fileID)
	if b.deleteFn != nil {
		return b.deleteFn(fileID)
	}
	return nil
}
