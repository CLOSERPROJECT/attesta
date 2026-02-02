package main

import (
	"context"
	"errors"
	"sort"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
}

type MongoStore struct {
	db *mongo.Database
}

func (s *MongoStore) InsertProcess(ctx context.Context, process Process) (primitive.ObjectID, error) {
	result, err := s.db.Collection("processes").InsertOne(ctx, process)
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
	if err := s.db.Collection("processes").FindOne(ctx, bson.M{"_id": id}).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) LoadLatestProcess(ctx context.Context) (*Process, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var process Process
	if err := s.db.Collection("processes").FindOne(ctx, bson.M{}, opts).Decode(&process); err != nil {
		return nil, err
	}
	return &process, nil
}

func (s *MongoStore) ListRecentProcesses(ctx context.Context, limit int64) ([]Process, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cursor, err := s.db.Collection("processes").Find(ctx, bson.M{}, opts)
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
	return s.db.Collection("processes").FindOneAndUpdate(ctx, bson.M{"_id": id}, update).Err()
}

func (s *MongoStore) UpdateProcessStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	_, err := s.db.Collection("processes").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status}})
	return err
}

func (s *MongoStore) InsertNotarization(ctx context.Context, notarization Notarization) error {
	_, err := s.db.Collection("notarizations").InsertOne(ctx, notarization)
	return err
}

type MemoryStore struct {
	mu            sync.RWMutex
	processes     map[primitive.ObjectID]Process
	notarizations []Notarization

	InsertProcessErr  error
	LoadProcessErr    error
	LoadLatestErr     error
	ListProcessesErr  error
	UpdateProgressErr error
	UpdateStatusErr   error
	InsertNotarizeErr error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{processes: map[primitive.ObjectID]Process{}}
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
