package main

import (
	"context"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionJoinRequests                 = "join_requests"
	collectionOrganizationCreationRequests = "organization_creation_requests"
)

func cloneJoinRequest(req JoinRequest) JoinRequest {
	cloned := req
	if req.RoleSlugs != nil {
		cloned.RoleSlugs = append([]string(nil), req.RoleSlugs...)
	}
	return cloned
}

func cloneOrganizationCreationRequest(req OrganizationCreationRequest) OrganizationCreationRequest {
	return req
}

func prepareJoinRequestForInsert(req JoinRequest, now time.Time) JoinRequest {
	if req.ID.IsZero() {
		req.ID = primitive.NewObjectID()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = req.CreatedAt
	}
	if req.Status == "" {
		req.Status = AffiliationStatusPending
	}
	if req.RoleSlugs != nil {
		req.RoleSlugs = append([]string(nil), req.RoleSlugs...)
	}
	return req
}

func prepareOrganizationCreationRequestForInsert(req OrganizationCreationRequest, now time.Time) OrganizationCreationRequest {
	if req.ID.IsZero() {
		req.ID = primitive.NewObjectID()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = req.CreatedAt
	}
	if req.Status == "" {
		req.Status = AffiliationStatusPending
	}
	return req
}

func (s *MemoryStore) ensureAffiliationMaps() {
	if s.joinRequests == nil {
		s.joinRequests = map[primitive.ObjectID]JoinRequest{}
	}
	if s.organizationCreationRequests == nil {
		s.organizationCreationRequests = map[primitive.ObjectID]OrganizationCreationRequest{}
	}
}

func (s *MemoryStore) InsertJoinRequest(_ context.Context, req JoinRequest) (JoinRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	req = prepareJoinRequestForInsert(req, time.Now().UTC())
	s.joinRequests[req.ID] = cloneJoinRequest(req)
	return cloneJoinRequest(req), nil
}

func (s *MemoryStore) LoadJoinRequestByID(_ context.Context, id primitive.ObjectID) (*JoinRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.joinRequests[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneJoinRequest(req)
	return &cloned, nil
}

func (s *MemoryStore) UpdateJoinRequest(_ context.Context, req JoinRequest) (JoinRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	if req.ID.IsZero() {
		return JoinRequest{}, mongo.ErrNoDocuments
	}
	if _, ok := s.joinRequests[req.ID]; !ok {
		return JoinRequest{}, mongo.ErrNoDocuments
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}
	s.joinRequests[req.ID] = cloneJoinRequest(req)
	return cloneJoinRequest(req), nil
}

func (s *MemoryStore) DeleteJoinRequest(_ context.Context, id primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	if id.IsZero() {
		return mongo.ErrNoDocuments
	}
	if _, ok := s.joinRequests[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.joinRequests, id)
	return nil
}

func (s *MemoryStore) ListPendingJoinRequestsByOrg(_ context.Context, orgSlug string) ([]JoinRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slug := strings.TrimSpace(orgSlug)
	items := make([]JoinRequest, 0)
	for _, req := range s.joinRequests {
		if req.Status != AffiliationStatusPending {
			continue
		}
		if slug != "" && req.OrgSlug != slug {
			continue
		}
		items = append(items, cloneJoinRequest(req))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID.Hex() > items[j].ID.Hex()
	})
	return items, nil
}

func (s *MemoryStore) FindPendingJoinRequestByUser(_ context.Context, userID string) (*JoinRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, nil
	}
	var found *JoinRequest
	for _, req := range s.joinRequests {
		if req.RequesterUserID != uid || req.Status != AffiliationStatusPending {
			continue
		}
		cloned := cloneJoinRequest(req)
		if found == nil || cloned.CreatedAt.After(found.CreatedAt) ||
			(cloned.CreatedAt.Equal(found.CreatedAt) && cloned.ID.Hex() > found.ID.Hex()) {
			found = &cloned
		}
	}
	return found, nil
}

func (s *MemoryStore) InsertOrganizationCreationRequest(_ context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	req = prepareOrganizationCreationRequestForInsert(req, time.Now().UTC())
	s.organizationCreationRequests[req.ID] = cloneOrganizationCreationRequest(req)
	return cloneOrganizationCreationRequest(req), nil
}

func (s *MemoryStore) LoadOrganizationCreationRequestByID(_ context.Context, id primitive.ObjectID) (*OrganizationCreationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.organizationCreationRequests[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	cloned := cloneOrganizationCreationRequest(req)
	return &cloned, nil
}

func (s *MemoryStore) UpdateOrganizationCreationRequest(_ context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	if req.ID.IsZero() {
		return OrganizationCreationRequest{}, mongo.ErrNoDocuments
	}
	if _, ok := s.organizationCreationRequests[req.ID]; !ok {
		return OrganizationCreationRequest{}, mongo.ErrNoDocuments
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}
	s.organizationCreationRequests[req.ID] = cloneOrganizationCreationRequest(req)
	return cloneOrganizationCreationRequest(req), nil
}

func (s *MemoryStore) DeleteOrganizationCreationRequest(_ context.Context, id primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureAffiliationMaps()
	if id.IsZero() {
		return mongo.ErrNoDocuments
	}
	if _, ok := s.organizationCreationRequests[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(s.organizationCreationRequests, id)
	return nil
}

func (s *MemoryStore) ListPendingOrganizationCreationRequests(_ context.Context) ([]OrganizationCreationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]OrganizationCreationRequest, 0)
	for _, req := range s.organizationCreationRequests {
		if req.Status != AffiliationStatusPending {
			continue
		}
		items = append(items, cloneOrganizationCreationRequest(req))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID.Hex() > items[j].ID.Hex()
	})
	return items, nil
}

func (s *MemoryStore) FindPendingOrganizationCreationRequestByUser(_ context.Context, userID string) (*OrganizationCreationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, nil
	}
	var found *OrganizationCreationRequest
	for _, req := range s.organizationCreationRequests {
		if req.RequesterUserID != uid || req.Status != AffiliationStatusPending {
			continue
		}
		cloned := cloneOrganizationCreationRequest(req)
		if found == nil || cloned.CreatedAt.After(found.CreatedAt) ||
			(cloned.CreatedAt.Equal(found.CreatedAt) && cloned.ID.Hex() > found.ID.Hex()) {
			found = &cloned
		}
	}
	return found, nil
}

func (s *MongoStore) InsertJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error) {
	req = prepareJoinRequestForInsert(req, time.Now().UTC())
	if _, err := s.database().Collection(collectionJoinRequests).InsertOne(ctx, req); err != nil {
		return JoinRequest{}, err
	}
	return req, nil
}

func (s *MongoStore) LoadJoinRequestByID(ctx context.Context, id primitive.ObjectID) (*JoinRequest, error) {
	var req JoinRequest
	if err := s.database().Collection(collectionJoinRequests).FindOne(ctx, bson.M{"_id": id}).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *MongoStore) UpdateJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error) {
	if req.ID.IsZero() {
		return JoinRequest{}, mongo.ErrNoDocuments
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}
	result, err := s.database().Collection(collectionJoinRequests).UpdateOne(
		ctx,
		bson.M{"_id": req.ID},
		bson.M{"$set": bson.M{
			"requesterUserId": req.RequesterUserID,
			"requesterEmail":  req.RequesterEmail,
			"orgSlug":         req.OrgSlug,
			"roleSlugs":       req.RoleSlugs,
			"status":          req.Status,
			"rejectReason":    req.RejectReason,
			"decidedByUserId": req.DecidedByUserID,
			"createdAt":       req.CreatedAt,
			"updatedAt":       req.UpdatedAt,
			"decidedAt":       req.DecidedAt,
		}},
	)
	if err != nil {
		return JoinRequest{}, err
	}
	if result != nil && result.MatchedCount == 0 {
		return JoinRequest{}, mongo.ErrNoDocuments
	}
	return req, nil
}

func (s *MongoStore) DeleteJoinRequest(ctx context.Context, id primitive.ObjectID) error {
	if id.IsZero() {
		return mongo.ErrNoDocuments
	}
	result, err := s.database().Collection(collectionJoinRequests).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result != nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) ListPendingJoinRequestsByOrg(ctx context.Context, orgSlug string) ([]JoinRequest, error) {
	filter := bson.M{"status": AffiliationStatusPending}
	if slug := strings.TrimSpace(orgSlug); slug != "" {
		filter["orgSlug"] = slug
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}})
	cursor, err := s.database().Collection(collectionJoinRequests).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]JoinRequest, 0)
	for cursor.Next(ctx) {
		var req JoinRequest
		if err := cursor.Decode(&req); err != nil {
			continue
		}
		items = append(items, req)
	}
	return items, cursor.Err()
}

func (s *MongoStore) FindPendingJoinRequestByUser(ctx context.Context, userID string) (*JoinRequest, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, nil
	}
	var req JoinRequest
	err := s.database().Collection(collectionJoinRequests).FindOne(
		ctx,
		bson.M{"requesterUserId": uid, "status": AffiliationStatusPending},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}),
	).Decode(&req)
	switch {
	case err == nil:
		return &req, nil
	case err == mongo.ErrNoDocuments:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *MongoStore) InsertOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	req = prepareOrganizationCreationRequestForInsert(req, time.Now().UTC())
	if _, err := s.database().Collection(collectionOrganizationCreationRequests).InsertOne(ctx, req); err != nil {
		return OrganizationCreationRequest{}, err
	}
	return req, nil
}

func (s *MongoStore) LoadOrganizationCreationRequestByID(ctx context.Context, id primitive.ObjectID) (*OrganizationCreationRequest, error) {
	var req OrganizationCreationRequest
	if err := s.database().Collection(collectionOrganizationCreationRequests).FindOne(ctx, bson.M{"_id": id}).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *MongoStore) UpdateOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	if req.ID.IsZero() {
		return OrganizationCreationRequest{}, mongo.ErrNoDocuments
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}
	result, err := s.database().Collection(collectionOrganizationCreationRequests).UpdateOne(
		ctx,
		bson.M{"_id": req.ID},
		bson.M{"$set": bson.M{
			"requesterUserId": req.RequesterUserID,
			"requesterEmail":  req.RequesterEmail,
			"proposedName":    req.ProposedName,
			"proposedSlug":    req.ProposedSlug,
			"status":          req.Status,
			"rejectReason":    req.RejectReason,
			"decidedByUserId": req.DecidedByUserID,
			"createdAt":       req.CreatedAt,
			"updatedAt":       req.UpdatedAt,
			"decidedAt":       req.DecidedAt,
		}},
	)
	if err != nil {
		return OrganizationCreationRequest{}, err
	}
	if result != nil && result.MatchedCount == 0 {
		return OrganizationCreationRequest{}, mongo.ErrNoDocuments
	}
	return req, nil
}

func (s *MongoStore) DeleteOrganizationCreationRequest(ctx context.Context, id primitive.ObjectID) error {
	if id.IsZero() {
		return mongo.ErrNoDocuments
	}
	result, err := s.database().Collection(collectionOrganizationCreationRequests).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result != nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) ListPendingOrganizationCreationRequests(ctx context.Context) ([]OrganizationCreationRequest, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}})
	cursor, err := s.database().Collection(collectionOrganizationCreationRequests).Find(
		ctx,
		bson.M{"status": AffiliationStatusPending},
		opts,
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]OrganizationCreationRequest, 0)
	for cursor.Next(ctx) {
		var req OrganizationCreationRequest
		if err := cursor.Decode(&req); err != nil {
			continue
		}
		items = append(items, req)
	}
	return items, cursor.Err()
}

func (s *MongoStore) FindPendingOrganizationCreationRequestByUser(ctx context.Context, userID string) (*OrganizationCreationRequest, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, nil
	}
	var req OrganizationCreationRequest
	err := s.database().Collection(collectionOrganizationCreationRequests).FindOne(
		ctx,
		bson.M{"requesterUserId": uid, "status": AffiliationStatusPending},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}),
	).Decode(&req)
	switch {
	case err == nil:
		return &req, nil
	case err == mongo.ErrNoDocuments:
		return nil, nil
	default:
		return nil, err
	}
}
