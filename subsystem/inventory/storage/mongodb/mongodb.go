// Package mongodb implements a MongoDB storage backend for the inventory subsystem.
package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	defaultURI        = "mongodb://localhost:27017"
	defaultDatabase   = "nanocmd"
	defaultCollection = "subsystem_inventory"
)

// MongoDBStorage implements inventory storage using MongoDB.
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
// Default collection is "subsystem_inventory".
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

type inventoryDocument struct {
	ID        string `bson:"_id"`
	RawValues []byte `bson:"raw_values"`
}

// RetrieveInventory queries and returns the inventory values mapped by enrollment ID from MongoDB.
// Must provide opt and IDs.
func (s *MongoDBStorage) RetrieveInventory(ctx context.Context, opt *storage.SearchOptions) (map[string]storage.Values, error) {
	if opt == nil || len(opt.IDs) < 1 {
		return nil, storage.ErrNoIDs
	}

	cursor, err := s.collection.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": opt.IDs}},
		options.Find().SetProjection(bson.M{"raw_values": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	ret := make(map[string]storage.Values)
	for cursor.Next(ctx) {
		var doc inventoryDocument
		if err := cursor.Decode(&doc); err != nil {
			return ret, err
		}
		var values storage.Values
		if err := json.Unmarshal(doc.RawValues, &values); err != nil {
			return ret, fmt.Errorf("unmarshal values for %s: %w", doc.ID, err)
		}
		ret[doc.ID] = values
	}
	if err := cursor.Err(); err != nil {
		return ret, err
	}

	return ret, nil
}

// StoreInventoryValues stores inventory data about the specified ID in MongoDB.
func (s *MongoDBStorage) StoreInventoryValues(ctx context.Context, id string, newValues storage.Values) error {
	if id == "" {
		return storage.ErrNoIDs
	}
	if len(newValues) == 0 {
		return nil
	}

	values := newValues
	var doc inventoryDocument
	err := s.collection.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"raw_values": 1})).Decode(&doc)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("get values: %w", err)
	}
	if len(doc.RawValues) > 0 {
		if err = json.Unmarshal(doc.RawValues, &values); err != nil {
			return fmt.Errorf("unmarshal values: %w", err)
		}
		for key, value := range newValues {
			values[key] = value
		}
	}

	rawValues, err := marshalValues(values)
	if err != nil {
		return err
	}
	_, err = s.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"raw_values": rawValues}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("set values: %w", err)
	}

	return nil
}

// DeleteInventory deletes all inventory data for an enrollment ID from MongoDB.
func (s *MongoDBStorage) DeleteInventory(ctx context.Context, id string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func marshalValues(values storage.Values) ([]byte, error) {
	jsonValues, err := json.Marshal(&values)
	if err != nil {
		return nil, fmt.Errorf("marshal values: %w", err)
	}
	return jsonValues, nil
}
