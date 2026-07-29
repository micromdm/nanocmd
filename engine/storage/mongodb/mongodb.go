// Package mongodb implements a MongoDB storage backend for the workflow engine.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/micromdm/nanocmd/engine/storage/kv"
	"github.com/micromdm/nanocmd/utils/uuid"
	nanolibkv "github.com/micromdm/nanolib/storage/kv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	defaultURI              = "mongodb://localhost:27017"
	defaultDatabase         = "nanocmd"
	defaultCollectionPrefix = "engine"
)

// MongoDBStorage implements workflow engine storage using MongoDB.
type MongoDBStorage struct {
	*kv.KV

	client *mongo.Client

	stepCollection   *mongo.Collection
	idCmdCollection  *mongo.Collection
	eventCollection  *mongo.Collection
	statusCollection *mongo.Collection
}

type config struct {
	uri              string
	database         string
	collectionPrefix string

	client *mongo.Client

	stepCollection   *mongo.Collection
	idCmdCollection  *mongo.Collection
	eventCollection  *mongo.Collection
	statusCollection *mongo.Collection
}

// Option allows configuring a MongoDBStorage.
type Option func(*config)

// WithURI sets the MongoDB connection URI.
//
// Default URI is "mongodb://localhost:27017".
// Value is ignored if WithClient or WithCollections is used.
func WithURI(uri string) Option {
	return func(c *config) {
		c.uri = uri
	}
}

// WithDatabase sets the MongoDB database name.
//
// Default database is "nanocmd".
// Value is ignored if WithCollections is used.
func WithDatabase(database string) Option {
	return func(c *config) {
		c.database = database
	}
}

// WithCollectionPrefix sets the prefix used for default engine collection names.
//
// The default collection names are "engine_steps", "engine_id_commands",
// "engine_event_subscriptions", and "engine_workflow_status".
// Value is ignored if WithCollections is used.
func WithCollectionPrefix(prefix string) Option {
	return func(c *config) {
		c.collectionPrefix = prefix
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

// WithCollections sets custom MongoDB collections for engine storage.
//
// If set, URI, database, collection prefix, and client options are ignored.
func WithCollections(step, idCmd, event, status *mongo.Collection) Option {
	return func(c *config) {
		c.stepCollection = step
		c.idCmdCollection = idCmd
		c.eventCollection = event
		c.statusCollection = status
	}
}

// New creates and returns a new MongoDBStorage.
func New(opts ...Option) (*MongoDBStorage, error) {
	cfg := &config{
		uri:              defaultURI,
		database:         defaultDatabase,
		collectionPrefix: defaultCollectionPrefix,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	collectionsConfigured := cfg.stepCollection != nil ||
		cfg.idCmdCollection != nil ||
		cfg.eventCollection != nil ||
		cfg.statusCollection != nil
	if collectionsConfigured {
		if cfg.stepCollection == nil || cfg.idCmdCollection == nil || cfg.eventCollection == nil || cfg.statusCollection == nil {
			return nil, errors.New("all mongodb engine collections are required")
		}
		cfg.client = cfg.stepCollection.Database().Client()
	} else {
		if cfg.database == "" {
			return nil, errors.New("mongodb database is required")
		}
		if cfg.collectionPrefix == "" {
			return nil, errors.New("mongodb collection prefix is required")
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

		db := cfg.client.Database(cfg.database)
		cfg.stepCollection = db.Collection(cfg.collectionPrefix + "_steps")
		cfg.idCmdCollection = db.Collection(cfg.collectionPrefix + "_id_commands")
		cfg.eventCollection = db.Collection(cfg.collectionPrefix + "_event_subscriptions")
		cfg.statusCollection = db.Collection(cfg.collectionPrefix + "_workflow_status")
	}

	return &MongoDBStorage{
		KV: kv.New(
			&bucket{collection: cfg.stepCollection},
			&bucket{collection: cfg.idCmdCollection},
			&bucket{collection: cfg.eventCollection},
			uuid.NewUUID(),
			&bucket{collection: cfg.statusCollection},
		),
		client:           cfg.client,
		stepCollection:   cfg.stepCollection,
		idCmdCollection:  cfg.idCmdCollection,
		eventCollection:  cfg.eventCollection,
		statusCollection: cfg.statusCollection,
	}, nil
}

type bucket struct {
	collection *mongo.Collection
}

type bucketDocument struct {
	Key   string `bson:"_id"`
	Value []byte `bson:"value"`
}

func (b *bucket) Get(ctx context.Context, key string) ([]byte, error) {
	var doc bucketDocument
	err := b.collection.FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("%w: %s", nanolibkv.ErrKeyNotFound, key)
	}
	if err != nil {
		return nil, err
	}
	return doc.Value, nil
}

func (b *bucket) Set(ctx context.Context, key string, value []byte) error {
	_, err := b.collection.UpdateOne(
		ctx,
		bson.M{"_id": key},
		bson.M{"$set": bson.M{"value": value}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (b *bucket) Has(ctx context.Context, key string) (bool, error) {
	err := b.collection.FindOne(
		ctx,
		bson.M{"_id": key},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func (b *bucket) Delete(ctx context.Context, key string) error {
	_, err := b.collection.DeleteOne(ctx, bson.M{"_id": key})
	return err
}

func (b *bucket) Keys(ctx context.Context, cancel <-chan struct{}) <-chan string {
	return b.KeysPrefix(ctx, "", cancel)
}

func (b *bucket) KeysPrefix(ctx context.Context, prefix string, cancel <-chan struct{}) <-chan string {
	keys := make(chan string)
	go func() {
		defer close(keys)

		cursor, err := b.collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
		if err != nil {
			return
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var doc bucketDocument
			if err := cursor.Decode(&doc); err != nil {
				return
			}
			if prefix != "" && !strings.HasPrefix(doc.Key, prefix) {
				continue
			}
			select {
			case <-cancel:
				return
			case keys <- doc.Key:
			}
		}
	}()
	return keys
}
