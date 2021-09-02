module github.com/amammay/effectivecloudrun

go 1.16

require (
	cloud.google.com/go v0.93.3
	cloud.google.com/go/firestore v1.5.0
	cloud.google.com/go/trace v0.1.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go v1.0.0-RC2.0.20210816152642-29dd0bfc39f0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.0.0-RC2.0.20210816152642-29dd0bfc39f0
	github.com/blendle/zapdriver v1.3.1
	github.com/brianvoe/gofakeit/v6 v6.7.1
	github.com/gorilla/mux v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.22.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.22.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.22.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
	go.opentelemetry.io/otel/trace v1.0.0-RC2
	go.uber.org/zap v1.19.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/api v0.54.0
	google.golang.org/grpc v1.39.1
)
