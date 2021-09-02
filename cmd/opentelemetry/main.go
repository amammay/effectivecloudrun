package main

import (
	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/amammay/effectivecloudrun/internal/logx"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	AppName = "opentelemetry"
)

type server struct {
	router    *mux.Router
	logger    *logx.AppLogger
	firestore *firestore.Client
	bin       *binClient
}

func (s *server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.router.ServeHTTP(writer, request)
}

func newServer(logger *logx.AppLogger, firestoreClient *firestore.Client, binClient *binClient) *server {
	s := &server{router: mux.NewRouter(), logger: logger, firestore: firestoreClient, bin: binClient}
	s.routes()
	return s
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("run(): %v", err)
	}
}

func run() error {
	// retrieves our project id from the gcp metadata server
	projectID := "mammay-labs"
	onGCE := metadata.OnGCE()
	if onGCE {
		id, err := metadata.ProjectID()
		if err != nil {
			return fmt.Errorf("metadata.ProjectID(): %v", err)
		}
		projectID = id
	}

	loggerClient, err := logx.NewLogger(projectID, onGCE)
	if err != nil {
		return fmt.Errorf("logx.NewLogger(): %v", err)
	}

	logger := loggerClient.Sugar()
	defer logger.Sync()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// setup tracing, defer the teardown of the tracer to flush it
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

	unaryInterceptor := grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor())
	streamInterceptor := grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor())
	firestoreClient, err := firestore.NewClient(ctx, projectID, option.WithGRPCDialOption(unaryInterceptor), option.WithGRPCDialOption(streamInterceptor))
	if err != nil {
		return fmt.Errorf("firestore.NewClient(): %v", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)
	binClient := NewBinClient(httpClient, "https://httpbin.org/")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: newServer(loggerClient, firestoreClient, binClient),
	}
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
	logger.Infof("starting server on %s", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("httpServer.ListenAndServe(): %v", err)
	}
	return g.Wait()
}
