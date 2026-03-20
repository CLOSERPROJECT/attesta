package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetUserByMongoID keeps legacy process history readable during the migration.
func (s *MongoStore) GetUserByMongoID(ctx context.Context, userMongoID primitive.ObjectID) (*AccountUser, error) {
	var user AccountUser
	if err := s.database().Collection("users").FindOne(ctx, bson.M{"_id": userMongoID}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
