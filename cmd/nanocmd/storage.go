package main

import (
	"fmt"
	"path/filepath"

	storageeng "github.com/micromdm/nanocmd/engine/storage"
	storageengdiskv "github.com/micromdm/nanocmd/engine/storage/diskv"
	storageenginmem "github.com/micromdm/nanocmd/engine/storage/inmem"
	storageengmysql "github.com/micromdm/nanocmd/engine/storage/mysql"
	storagecmdplan "github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	storagecmdplandiskv "github.com/micromdm/nanocmd/subsystem/cmdplan/storage/diskv"
	storagecmdplaninmem "github.com/micromdm/nanocmd/subsystem/cmdplan/storage/inmem"
	storagefv "github.com/micromdm/nanocmd/subsystem/filevault/storage"
	storagefvdiskv "github.com/micromdm/nanocmd/subsystem/filevault/storage/diskv"
	storagefvinmem "github.com/micromdm/nanocmd/subsystem/filevault/storage/inmem"
	storagefvinvprk "github.com/micromdm/nanocmd/subsystem/filevault/storage/invprk"
	storageinv "github.com/micromdm/nanocmd/subsystem/inventory/storage"
	storageinvdiskv "github.com/micromdm/nanocmd/subsystem/inventory/storage/diskv"
	storageinvinmem "github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
	storageprof "github.com/micromdm/nanocmd/subsystem/profile/storage"
	storageprofdiskv "github.com/micromdm/nanocmd/subsystem/profile/storage/diskv"
	storageprofinmem "github.com/micromdm/nanocmd/subsystem/profile/storage/inmem"
	storageprofmysql "github.com/micromdm/nanocmd/subsystem/profile/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
)

type storageConfig struct {
	inventory storageinv.Storage
	engine    storageeng.AllStorage
	profile   storageprof.Storage
	cmdplan   storagecmdplan.Storage
	event     storageeng.EventSubscriptionStorage
	filevault storagefv.FVRotate
}

func parseStorage(name, dsn, _ string) (*storageConfig, error) {
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
