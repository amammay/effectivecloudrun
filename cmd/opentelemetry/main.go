package main

import (
	"cloud.google.com/go/compute/metadata"
	"context"
	"fmt"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	cloudprop "github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator"
	"github.com/blendle/zapdriver"
	"go.opentelemetry.io/otel"
	prop "go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type teardown func() error

func main() {
	if err := run(); err != nil {
		log.Fatalf("run(): %v", err)
	}
}

func run() error {
	// retrieves our project id from the gcp metadata server
	projectID := ""
	onGCE := metadata.OnGCE()
	if onGCE {
		id, err := metadata.ProjectID()
		if err != nil {
			return fmt.Errorf("metadata.ProjectID(): %v", err)
		}
		projectID = id
	}

	// create our uber zap configuration
	config := zapdriver.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	clientLogger, err := config.Build()
	if err != nil {
		return fmt.Errorf("config.Build(): %v", err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	logger := clientLogger.Sugar()
	tracingTeardown, err := initTracing(ctx, logger, projectID)
	if err != nil {
		return fmt.Errorf("initTracing(): %v", err)
	}
	defer func() {
		err := tracingTeardown()
		if err != nil {
			logger.Errorf("tracingTeardown(): %v", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: nil,
	}
	httpServer.RegisterOnShutdown(cancelFunc)
	// setup our shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(
		shutdown,
		os.Interrupt,    // Capture ctrl + c events
		syscall.SIGTERM, // Capture actual sig term event (kill command).
	)

	// setup our errgroup is listen for shutdown signal, from there attempt to shutdown our http server and capture any errors during shutdown
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		o := <-shutdown
		logger.Infof("sig: %s - starting shutting down sequence...", o)
		// we need to use a fresh context.Background() because the parent ctx we have in our current scope will be cancelled during the Shutdown method call
		graceFull, cancel := context.WithTimeout(context.Background(), 9*time.Second)
		defer cancel()
		// Shutdown the server with a timeout
		if err := httpServer.Shutdown(graceFull); err != nil {
			return fmt.Errorf("httpServer.Shutdown(): %w", err)
		}
		logger.Info("server has shutdown gracefully")
		return nil
	})
	logger.Infof("starting server on %q", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("httpServer.ListenAndServe(): %v", err)
	}
	return g.Wait()
}

type errorProcessing struct {
	logger *zap.SugaredLogger
}

func (e *errorProcessing) Handle(err error) {
	if err != nil {
		e.logger.Errorw("global otel error detected", "error", err)
	}
}

// initTracing will setup open telemetry with exporting results directly to gcp
func initTracing(ctx context.Context, logger *zap.SugaredLogger, projectID string) (teardown, error) {

	// set an error handler to bubble up any errors that otel might throw
	otel.SetErrorHandler(&errorProcessing{logger: logger})

	// set a text map propagator that is able to parse a variety of http headers, in our case CloudTraceFormatPropagator will handle
	// the header of X-Cloud-Trace-Context that gcp will set from the GFE
	otel.SetTextMapPropagator(prop.NewCompositeTextMapPropagator(
		cloudprop.CloudTraceFormatPropagator{},
		prop.TraceContext{},
		prop.Baggage{},
	))

	// create cloudtrace exporter
	exporter, err := cloudtrace.New(cloudtrace.WithProjectID(projectID), cloudtrace.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("cloudtrace.New(): %v", err)
	}

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(batchSpanProcessor))
	otel.SetTracerProvider(tp)

	return func() error {
		err := tp.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("tp.Shutdown(): %v", err)
		}
		return nil
	}, nil
}
