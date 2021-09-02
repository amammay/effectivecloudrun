package main

import (
	"context"
	"fmt"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	cloudprop "github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	prop "go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	instrumentationName = "github.com/amammay/effectivecloudrun/cmd/opentelemetry"
)

type teardown func() error

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

	exporter, err := cloudtrace.New(cloudtrace.WithProjectID(projectID), cloudtrace.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("cloudtrace.New(): %v", err)
	}

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(batchSpanProcessor), sdktrace.WithResource(
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(AppName),
			attribute.String("exporter", "google-cloud"),
		),
	))
	otel.SetTracerProvider(tp)

	return func() error {
		err := tp.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("tp.Shutdown(): %v", err)
		}
		return nil
	}, nil
}

func startSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.GetTracerProvider().Tracer(instrumentationName).Start(ctx, name, opts...)
}
