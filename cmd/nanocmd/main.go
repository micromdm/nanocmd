// Package main starts a NanoCMD server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/micromdm/nanocmd/engine"
	enginehttp "github.com/micromdm/nanocmd/engine/http"
	httpcmd "github.com/micromdm/nanocmd/http"
	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/mdm/foss"
	cmdplanhttp "github.com/micromdm/nanocmd/subsystem/cmdplan/http"
	fvenablehttp "github.com/micromdm/nanocmd/subsystem/filevault/http"
	invhttp "github.com/micromdm/nanocmd/subsystem/inventory/http"
	profhttp "github.com/micromdm/nanocmd/subsystem/profile/http"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/envflag"
	nanohttp "github.com/micromdm/nanolib/http"
	"github.com/micromdm/nanolib/http/trace"
	"github.com/micromdm/nanolib/log/stdlogfmt"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	apiUsername = "nanocmd"
	apiRealm    = "nanocmd"
)

func main() {
	var (
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flListen  = flag.String("listen", ":9003", "HTTP listen address")
		flVersion = flag.Bool("version", false, "print version and exit")
		flDumpWH  = flag.Bool("dump-webhook", false, "dump webhook input")
		flAPIKey  = flag.String("api", "", "API key for API endpoints")
		flEnqURL  = flag.String("enqueue-url", "", "URL of MDM server enqueue endpoint")
		flPushURL = flag.String("push-url", "", "URL of MDM server push endpoint")
		flEnqAPI  = flag.String("enqueue-api", "", "MDM server API key")
		flStorage = flag.String("storage", "file", "name of storage backend")
		flDSN     = flag.String("storage-dsn", "", "data source name (e.g. connection string or path)")
		flOptions = flag.String("storage-options", "", "storage backend options")
		flMicro   = flag.Bool("micromdm", false, "MicroMDM-style command submission")
		flWorkSec = flag.Uint("worker-interval", uint(engine.DefaultDuration/time.Second), "interval for worker in seconds")
		flPushSec = flag.Uint("repush-interval", uint(engine.DefaultRePushDuration/time.Second), "interval for repushes in seconds")
		flStTOSec = flag.Uint("step-timeout", uint(engine.DefaultTimeout/time.Second), "default step timeout in seconds")
	)
	envflag.Parse("NANOCMD_", []string{"version"})

	if *flVersion {
		fmt.Println(version)
		return
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	if *flEnqURL == "" || *flEnqAPI == "" || *flPushURL == "" {
		logger.Info(logkeys.Error, "enqueue URL, push URL, and API required")
		os.Exit(1)
	}

	// configure storage
	storage, err := parseStorage(*flStorage, *flDSN, *flOptions)
	if err != nil {
		logger.Info(logkeys.Message, "parse storage", logkeys.Error, err)
		os.Exit(1)
	}

	// configure our "MDM" i.e. how we send commands and receive responses
	opts := []foss.Option{
		foss.WithLogger(logger.With("service", "mdm")),
		foss.WithPush(*flPushURL),
	}
	if *flMicro {
		opts = append(opts, foss.WithMicroMDM())
	}
	fossMDM, err := foss.NewFossMDM(*flEnqURL, *flEnqAPI, opts...)
	if err != nil {
		logger.Info(logkeys.Message, "creating enqueuer", logkeys.Error, err)
		os.Exit(1)
	}

	// configure the workflow engine
	eOpts := []engine.Option{engine.WithLogger(logger.With("service", "engine"))}
	if *flStTOSec > 0 {
		eOpts = append(eOpts, engine.WithDefaultTimeout(time.Second*time.Duration(*flStTOSec)))
	}
	if storage.event != nil {
		eOpts = append(eOpts, engine.WithEventStorage(storage.event))
	}
	e := engine.New(storage.engine, fossMDM, eOpts...)

	// configure the workflow engine worker (async runner/job)
	var eWorker *engine.Worker
	if *flWorkSec > 0 {
		wOpts := []engine.WorkerOption{
			engine.WithWorkerLogger(logger.With("service", "engine worker")),
			engine.WithWorkerDuration(time.Second * time.Duration(*flWorkSec)),
		}
		if *flPushSec > 0 {
			wOpts = append(wOpts, engine.WithWorkerRePushDuration(time.Second*time.Duration(*flPushSec)))
		}
		eWorker = engine.NewWorker(
			e,
			storage.engine,
			fossMDM,
			wOpts...,
		)
	}

	// register workflows with the engine
	err = registerWorkflows(logger, e, storage, e)
	if err != nil {
		logger.Info(logkeys.Message, "registering workflows", logkeys.Error, err)
		os.Exit(1)
	}

	mux := flow.New()

	mux.Handle("/version", nanohttp.NewJSONVersionHandler(version))

	var eventHandler foss.MDMEventReceiver = e
	if *flDumpWH {
		eventHandler = foss.NewMDMEventDumper(eventHandler, os.Stdout)
	}
	var h http.Handler = foss.WebhookHandler(eventHandler, logger.With("handler", "webhook"))
	if *flDumpWH {
		h = httpcmd.DumpHandler(h, os.Stdout)
	}

	mux.Handle("/webhook", h)

	if *flAPIKey != "" {
		mux.Group(func(mux *flow.Mux) {
			mux.Use(func(h http.Handler) http.Handler {
				return nanohttp.NewSimpleBasicAuthHandler(h, apiUsername, *flAPIKey, apiRealm)
			})

			enginehttp.HandleAPIv1("/v1", mux, logger, e, storage.event)
			invhttp.HandleAPIv1("/v1", mux, logger, storage.inventory)
			profhttp.HandleAPIv1("/v1", mux, logger, storage.profile)
			fvenablehttp.HandleAPIv1("/v1", mux)
			cmdplanhttp.HandleAPIv1("/v1", mux, logger, storage.cmdplan)
		})
	}

	if eWorker != nil {
		go func() {
			err := eWorker.Run(context.Background())
			logs := []interface{}{logkeys.Message, "engine worker stopped"}
			if err != nil {
				logger.Info(append(logs, logkeys.Error, err)...)
				return
			}
			logger.Debug(logs)
		}()
	}

	// seed for newTraceID
	rand.Seed(time.Now().UnixNano())

	logger.Info(logkeys.Message, "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, trace.NewTraceLoggingHandler(mux, logger.With("handler", "log"), newTraceID))
	logs := []interface{}{logkeys.Message, "server shutdown"}
	if err != nil {
		logs = append(logs, logkeys.Error, err)
	}
	logger.Info(logs...)
}

type NullHandler struct{}

func (h *NullHandler) WebhookConnectEvent(ctx context.Context, id string, uuid string, raw []byte) error {
	return errors.New("[*NullHandler WebhookConnectEvent] not implemented")
}

// newTraceID generates a new HTTP trace ID for context logging.
// Currently this just makes a random string. This would be better
// served by e.g. https://github.com/oklog/ulid or something like
// https://opentelemetry.io/ someday.
func newTraceID(_ *http.Request) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
