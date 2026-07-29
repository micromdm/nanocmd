package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	storageeng "github.com/micromdm/nanocmd/engine/storage"
	storageengdiskv "github.com/micromdm/nanocmd/engine/storage/diskv"
	storageenginmem "github.com/micromdm/nanocmd/engine/storage/inmem"
	storageengmongodb "github.com/micromdm/nanocmd/engine/storage/mongodb"
	storageengmysql "github.com/micromdm/nanocmd/engine/storage/mysql"
	storagecmdplan "github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	storagecmdplandiskv "github.com/micromdm/nanocmd/subsystem/cmdplan/storage/diskv"
	storagecmdplaninmem "github.com/micromdm/nanocmd/subsystem/cmdplan/storage/inmem"
	storagecmdplanmongodb "github.com/micromdm/nanocmd/subsystem/cmdplan/storage/mongodb"
	storagefv "github.com/micromdm/nanocmd/subsystem/filevault/storage"
	storagefvdiskv "github.com/micromdm/nanocmd/subsystem/filevault/storage/diskv"
	storagefvinmem "github.com/micromdm/nanocmd/subsystem/filevault/storage/inmem"
	storagefvinvprk "github.com/micromdm/nanocmd/subsystem/filevault/storage/invprk"
	storagefvmongodb "github.com/micromdm/nanocmd/subsystem/filevault/storage/mongodb"
	storageinv "github.com/micromdm/nanocmd/subsystem/inventory/storage"
	storageinvdiskv "github.com/micromdm/nanocmd/subsystem/inventory/storage/diskv"
	storageinvinmem "github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
	storageinvmongodb "github.com/micromdm/nanocmd/subsystem/inventory/storage/mongodb"
	storageprof "github.com/micromdm/nanocmd/subsystem/profile/storage"
	storageprofdiskv "github.com/micromdm/nanocmd/subsystem/profile/storage/diskv"
	storageprofinmem "github.com/micromdm/nanocmd/subsystem/profile/storage/inmem"
	storageprofmongodb "github.com/micromdm/nanocmd/subsystem/profile/storage/mongodb"
	storageprofmysql "github.com/micromdm/nanocmd/subsystem/profile/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type storageConfig struct {
	inventory storageinv.Storage
	engine    storageeng.AllStorage
	profile   storageprof.Storage
	cmdplan   storagecmdplan.Storage
	event     storageeng.EventSubscriptionStorage
	filevault storagefv.FVRotate
}

func parseStorage(name, dsn, storageOptions string) (*storageConfig, error) {
	switch name {
	case "inmem":
		inv := storageinvinmem.New()
		fv, err := storagefvinmem.New(storagefvinvprk.NewInvPRK(inv))
		if err != nil {
			return nil, fmt.Errorf("creating filevault inmem storage: %w", err)
		}
		eng := storageenginmem.New()
		return &storageConfig{
			engine:    eng,
			inventory: inv,
			profile:   storageprofinmem.New(),
			cmdplan:   storagecmdplaninmem.New(),
			event:     eng,
			filevault: fv,
		}, nil
	case "file", "diskv":
		if dsn == "" {
			dsn = "db"
		}
		inv := storageinvdiskv.New(filepath.Join(dsn, "inventory"))
		fv, err := storagefvdiskv.New(filepath.Join(dsn, "fvkey"), storagefvinvprk.NewInvPRK(inv))
		if err != nil {
			return nil, fmt.Errorf("creating filevault diskv storage: %w", err)
		}
		eng := storageengdiskv.New(dsn)
		return &storageConfig{
			engine:    eng,
			inventory: inv,
			profile:   storageprofdiskv.New(filepath.Join(dsn, "profile")),
			cmdplan:   storagecmdplandiskv.New(filepath.Join(dsn, "cmdplan")),
			event:     eng,
			filevault: fv,
		}, nil
	case "mongodb":
		mongoOptions, err := parseMongoDBStorageOptions(storageOptions)
		if err != nil {
			return nil, err
		}

		uri := dsn
		if uri == "" {
			uri = "mongodb://localhost:27017"
		}
		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			return nil, fmt.Errorf("connecting to mongodb: %w", err)
		}
		if err = client.Ping(context.Background(), readpref.Primary()); err != nil {
			return nil, fmt.Errorf("pinging mongodb: %w", err)
		}
		if mongoOptions.database == "" {
			mongoOptions.database, err = mongoDBDatabaseFromURI(uri)
			if err != nil {
				return nil, err
			}
		}

		engOpts := []storageengmongodb.Option{storageengmongodb.WithClient(client)}
		invOpts := []storageinvmongodb.Option{storageinvmongodb.WithClient(client)}
		profOpts := []storageprofmongodb.Option{storageprofmongodb.WithClient(client)}
		cmdplanOpts := []storagecmdplanmongodb.Option{storagecmdplanmongodb.WithClient(client)}
		fvOpts := []storagefvmongodb.Option{storagefvmongodb.WithClient(client)}
		if mongoOptions.database != "" {
			engOpts = append(engOpts, storageengmongodb.WithDatabase(mongoOptions.database))
			invOpts = append(invOpts, storageinvmongodb.WithDatabase(mongoOptions.database))
			profOpts = append(profOpts, storageprofmongodb.WithDatabase(mongoOptions.database))
			cmdplanOpts = append(cmdplanOpts, storagecmdplanmongodb.WithDatabase(mongoOptions.database))
			fvOpts = append(fvOpts, storagefvmongodb.WithDatabase(mongoOptions.database))
		}
		if mongoOptions.collectionPrefix != "" {
			engOpts = append(engOpts, storageengmongodb.WithCollectionPrefix(mongoDBCollectionName(mongoOptions.collectionPrefix, "engine")))
			invOpts = append(invOpts, storageinvmongodb.WithCollectionName(mongoDBCollectionName(mongoOptions.collectionPrefix, "subsystem_inventory")))
			profOpts = append(profOpts, storageprofmongodb.WithCollectionName(mongoDBCollectionName(mongoOptions.collectionPrefix, "subsystem_profiles")))
			cmdplanOpts = append(cmdplanOpts, storagecmdplanmongodb.WithCollectionName(mongoDBCollectionName(mongoOptions.collectionPrefix, "subsystem_cmdplans")))
			fvOpts = append(fvOpts, storagefvmongodb.WithCollectionName(mongoDBCollectionName(mongoOptions.collectionPrefix, "subsystem_filevault")))
		}

		eng, err := storageengmongodb.New(engOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating engine mongodb storage: %w", err)
		}
		inv, err := storageinvmongodb.New(invOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating inventory mongodb storage: %w", err)
		}
		prof, err := storageprofmongodb.New(profOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating profile mongodb storage: %w", err)
		}
		cmdplan, err := storagecmdplanmongodb.New(cmdplanOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating cmdplan mongodb storage: %w", err)
		}
		fv, err := storagefvmongodb.New(storagefvinvprk.NewInvPRK(inv), fvOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating filevault mongodb storage: %w", err)
		}
		return &storageConfig{
			engine:    eng,
			inventory: inv,
			profile:   prof,
			cmdplan:   cmdplan,
			event:     eng,
			filevault: fv,
		}, nil
	case "mysql":
		inv := storageinvinmem.New()
		fv, err := storagefvinmem.New(storagefvinvprk.NewInvPRK(inv))
		if err != nil {
			return nil, fmt.Errorf("creating filevault inmem storage: %w", err)
		}
		eng, err := storageengmysql.New(storageengmysql.WithDSN(dsn))
		if err != nil {
			return nil, err
		}
		prof, err := storageprofmysql.New(storageprofmysql.WithDSN(dsn))
		if err != nil {
			return nil, err
		}
		return &storageConfig{
			engine:    eng,
			inventory: inv,
			profile:   prof,
			cmdplan:   storagecmdplaninmem.New(),
			event:     eng,
			filevault: fv,
		}, nil
	}
	return nil, fmt.Errorf("unknown storage: %s", name)
}

type mongoDBStorageOptions struct {
	database         string
	collectionPrefix string
}

func parseMongoDBStorageOptions(options string) (*mongoDBStorageOptions, error) {
	parsed := &mongoDBStorageOptions{}
	if options == "" {
		return parsed, nil
	}

	for _, option := range strings.Split(options, ",") {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}

		key, value, ok := strings.Cut(option, "=")
		if !ok {
			return nil, fmt.Errorf("invalid mongodb storage option: %s", option)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "database", "db":
			parsed.database = value
		case "collection_prefix":
			parsed.collectionPrefix = value
		default:
			return nil, fmt.Errorf("unknown mongodb storage option: %s", key)
		}
	}

	return parsed, nil
}

func mongoDBCollectionName(prefix, collection string) string {
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), "_")
	if prefix == "" {
		return collection
	}
	return prefix + "_" + collection
}

func mongoDBDatabaseFromURI(uri string) (string, error) {
	if uri == "" {
		return "", nil
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parsing mongodb URI: %w", err)
	}
	db := strings.Trim(u.EscapedPath(), "/")
	if db == "" {
		return "", nil
	}
	if idx := strings.Index(db, "/"); idx >= 0 {
		db = db[:idx]
	}
	db, err = url.PathUnescape(db)
	if err != nil {
		return "", fmt.Errorf("parsing mongodb database name: %w", err)
	}
	return db, nil
}
