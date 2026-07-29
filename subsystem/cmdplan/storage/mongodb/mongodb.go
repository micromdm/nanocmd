// Package mongodb implements a MongoDB storage backend for the Command Plan subsystem.
package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	nanolibkv "github.com/micromdm/nanolib/storage/kv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	defaultURI        = "mongodb://localhost:27017"
	defaultDatabase   = "nanocmd"
	defaultCollection = "subsystem_cmdplans"
)

// MongoDBStorage implements command plan storage using MongoDB.
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
// Default collection is "subsystem_cmdplans".
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

type cmdPlanDocument struct {
	Name string `bson:"_id"`
	Raw  []byte `bson:"raw_plan"`
}

// RetrieveCMDPlan unmarshals the JSON stored using name and returns the command plan.
func (s *MongoDBStorage) RetrieveCMDPlan(ctx context.Context, name string) (*storage.CMDPlan, error) {
	var doc cmdPlanDocument
	err := s.collection.FindOne(
		ctx,
		bson.M{"_id": name},
		options.FindOne().SetProjection(bson.M{"raw_plan": 1}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("%w: %s", nanolibkv.ErrKeyNotFound, name)
	}
	if err != nil {
		return nil, err
	}

	cmdPlan := new(storage.CMDPlan)
	return cmdPlan, json.Unmarshal(doc.Raw, cmdPlan)
}

// StoreCMDPlan marshals p into JSON and stores it using name.
func (s *MongoDBStorage) StoreCMDPlan(ctx context.Context, name string, p *storage.CMDPlan) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = s.collection.UpdateOne(
		ctx,
		bson.M{"_id": name},
		bson.M{"$set": bson.M{"raw_plan": raw}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// DeleteCMDPlan deletes the JSON stored using name.
func (s *MongoDBStorage) DeleteCMDPlan(ctx context.Context, name string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": name})
	return err
}
