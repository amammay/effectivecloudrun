package main

import (
	"cloud.google.com/go/compute/metadata"
	"fmt"
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"os"
)

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

	mux := http.NewServeMux()
	mux.HandleFunc("/stdlogger", stdlogger())
	mux.HandleFunc("/structuredlogger", structuredlogger(projectID))
	mux.HandleFunc("/uberzaplogger", uberzaplogger(projectID, onGCE))

	// cloud run sets the PORT env variable for us to listen on
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("starting server on %q", port)
	return http.ListenAndServe(":"+port, mux)
}

// stdlogger showcases the most basic of loggers that is included with golang, better then having nothing
func stdlogger() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("hello %q! im an standard logger from the golang standard library", request.UserAgent())
		fmt.Fprintf(writer, "<h1> howdy i am an standard logger %q", request.UserAgent())
	}
}

// structuredlogger showcases how we can optimize logging for google cloud to get more bang for our buck when writing logs
// there is definitely opportunities for natural abstractions to arise with this, therefore allowing teams to have full control if needed
func structuredlogger(projectID string) http.HandlerFunc {

	return func(writer http.ResponseWriter, request *http.Request) {

		debug(request, "debug message", projectID)
		info(request, "information message", projectID)
		notice(request, "notice message", projectID)
		warning(request, "warning message", projectID)
		errorl(request, "error message", projectID)
		critical(request, "critical message", projectID)
		alert(request, "alert message", projectID)
		emergency(request, "emergency message", projectID)

		fmt.Fprintf(writer, "<h1> structured logger is saying hello %q", request.UserAgent())

	}
}

// uberzaplogger showcases how using a third party logger introduces various quality of life updates from the structuredlogger
// the only downside is that its another third party library you are learning. overall the api surface is pretty straight forward with uber-zap
// we are just using a wrapper around zap to provide the correct configurations for gcp logging.
func uberzaplogger(projectID string, onGCE bool) http.HandlerFunc {

	var config zap.Config
	// if on the cloud we will use a production config
	if onGCE {
		// create our uber zap configuration
		config = zapdriver.NewProductionConfig()
		// set the min logging level to debug for this demo
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		// running locally we will use a human-readable output
		config = zapdriver.NewDevelopmentConfig()
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// creates our logger instance
	clientLogger, err := config.Build()
	if err != nil {
		log.Fatalf("zap.config.Build(): %v", err)
	}

	wrapTraceContext := func(header string) *zap.SugaredLogger {
		traceID, spanID, sampled := deconstructXCloudTraceContext(header)
		fields := zapdriver.TraceContext(traceID, spanID, sampled, projectID)
		setFields := clientLogger.With(fields...)
		return setFields.Sugar()
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		logger := wrapTraceContext(request.Header.Get("X-Cloud-Trace-Context"))
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
		// calling any below will cause application to quite from the behavior of uber zap
		// logger.DPanic("critical message")
		// logger.Panic("alert message")
		// logger.Fatal("EMERGENCY message")

		fmt.Fprintf(writer, "<h1> uber zap is saying hello %q", request.UserAgent())

	}
}
