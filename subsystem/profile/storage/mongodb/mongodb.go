// Package mongodb implements a MongoDB storage backend for the Profile subsystem.
package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	defaultURI        = "mongodb://localhost:27017"
	defaultDatabase   = "nanocmd"
	defaultCollection = "subsystem_profiles"
)

// MongoDBStorage implements profile storage using MongoDB.
type MongoDBStorage struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type config struct {
	uri            string
	database       string
	collectionName string

	client     *mongo.Client
	collection *mongo.Collection
}

// Option allows configuring a MongoDBStorage.
type Option func(*config)

// WithURI sets the MongoDB connection URI.
//
// Default URI is "mongodb://localhost:27017".
// Value is ignored if WithClient or WithCollection is used.
func WithURI(uri string) Option {
	return func(c *config) {
		c.uri = uri
	}
}

// WithDatabase sets the MongoDB database name.
//
// Default database is "nanocmd".
// Value is ignored if WithCollection is used.
func WithDatabase(database string) Option {
	return func(c *config) {
		c.database = database
	}
}

// WithCollectionName sets the MongoDB collection name.
//
// Default collection is "subsystem_profiles".
// Value is ignored if WithCollection is used.
func WithCollectionName(collection string) Option {
	return func(c *config) {
		c.collectionName = collection
	}
}

// WithClient sets a custom MongoDB client.
//
// If set, the URI passed via WithURI is ignored.
func WithClient(client *mongo.Client) Option {
	return func(c *config) {
		c.client = client
	}
}

// WithCollection sets a custom MongoDB collection.
//
// If set, URI, database, collection name, and client options are ignored.
func WithCollection(collection *mongo.Collection) Option {
	return func(c *config) {
		c.collection = collection
	}
}

// New creates and returns a new MongoDBStorage.
func New(opts ...Option) (*MongoDBStorage, error) {
	cfg := &config{
		uri:            defaultURI,
		database:       defaultDatabase,
		collectionName: defaultCollection,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.collection != nil {
		return &MongoDBStorage{
			client:     cfg.collection.Database().Client(),
			collection: cfg.collection,
		}, nil
	}

	if cfg.database == "" {
		return nil, errors.New("mongodb database is required")
	}
	if cfg.collectionName == "" {
		return nil, errors.New("mongodb collection is required")
	}

	var err error
	if cfg.client == nil {
		if cfg.uri == "" {
			return nil, errors.New("mongodb URI is required")
		}
		cfg.client, err = mongo.Connect(options.Client().ApplyURI(cfg.uri))
		if err != nil {
			return nil, err
		}
	}
	if err = cfg.client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, err
	}

	return &MongoDBStorage{
		client:     cfg.client,
		collection: cfg.client.Database(cfg.database).Collection(cfg.collectionName),
	}, nil
}

type profileDocument struct {
	Name        string `bson:"_id"`
	ProfileID   string `bson:"profile_id"`
	ProfileUUID string `bson:"profile_uuid"`
	RawProfile  []byte `bson:"raw_profile,omitempty"`
}

// RetrieveRawProfiles returns the raw profile bytes by name from MongoDB.
// Implementations should not return all profiles if no names were provided.
// ErrProfileNotFound is returned for any name that hasn't been stored.
// ErrNoNames is returned if names is empty.
func (s *MongoDBStorage) RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error) {
	if len(names) < 1 {
		return nil, storage.ErrNoNames
	}

	cursor, err := s.collection.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": names}},
		options.Find().SetProjection(bson.M{"raw_profile": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	ret := make(map[string][]byte)
	for cursor.Next(ctx) {
		var dbProfile profileDocument
		if err := cursor.Decode(&dbProfile); err != nil {
			return ret, err
		}
		ret[dbProfile.Name] = dbProfile.RawProfile
	}
	if err := cursor.Err(); err != nil {
		return ret, err
	}

	for _, name := range names {
		_, ok := ret[name]
		if !ok {
			return ret, fmt.Errorf("%w: %s: missing from result set", storage.ErrProfileNotFound, name)
		}
	}

	return ret, nil
}

// RetrieveProfileInfos returns the profile metadata by name from MongoDB.
// ErrProfileNotFound is returned for any name that hasn't been stored.
func (s *MongoDBStorage) RetrieveProfileInfos(ctx context.Context, names []string) (map[string]storage.ProfileInfo, error) {
	filter := bson.M{}
	if len(names) > 0 {
		filter = bson.M{"_id": bson.M{"$in": names}}
	}

	cursor, err := s.collection.Find(
		ctx,
		filter,
		options.Find().SetProjection(bson.M{"profile_id": 1, "profile_uuid": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	ret := make(map[string]storage.ProfileInfo)
	for cursor.Next(ctx) {
		var dbProfile profileDocument
		if err := cursor.Decode(&dbProfile); err != nil {
			return ret, err
		}
		ret[dbProfile.Name] = storage.ProfileInfo{
			Identifier: dbProfile.ProfileID,
			UUID:       dbProfile.ProfileUUID,
		}
	}
	if err := cursor.Err(); err != nil {
		return ret, err
	}

	for _, name := range names {
		_, ok := ret[name]
		if !ok {
			return ret, fmt.Errorf("%w: %s: missing from result set", storage.ErrProfileNotFound, name)
		}
	}

	return ret, nil
}

// StoreProfile stores a raw profile and associated info in profile storage by name from MongoDB.
// It is up to the caller to make sure info is correctly populated and matches the raw profile bytes.
func (s *MongoDBStorage) StoreProfile(ctx context.Context, name string, info storage.ProfileInfo, raw []byte) error {
	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": name},
		bson.M{
			"$set": bson.M{
				"profile_id":   info.Identifier,
				"profile_uuid": info.UUID,
				"raw_profile":  raw,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// DeleteProfile deletes a profile from profile storage by name from MongoDB.
// ErrProfileNotFound is returned for a name that hasn't been stored.
func (s *MongoDBStorage) DeleteProfile(ctx context.Context, name string) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": name})
	if err != nil {
		return err
	}
	if result.DeletedCount < 1 {
		return fmt.Errorf("%w: %s: missing from result set", storage.ErrProfileNotFound, name)
	}
	return nil
}
