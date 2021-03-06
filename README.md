# OpenTelemetry-Go

[![Circle CI](https://circleci.com/gh/open-telemetry/opentelemetry-go.svg?style=svg)](https://circleci.com/gh/open-telemetry/opentelemetry-go)
[![Docs](https://godoc.org/go.opentelemetry.io/otel?status.svg)](https://godoc.org/go.opentelemetry.io/otel)
[![Go Report Card](https://goreportcard.com/badge/go.opentelemetry.io/otel)](https://goreportcard.com/report/go.opentelemetry.io/otel)
[![Gitter](https://badges.gitter.im/open-telemetry/opentelemetry-go.svg)](https://gitter.im/open-telemetry/opentelemetry-go?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

The Go [OpenTelemetry](https://opentelemetry.io/) client.

## Installation

This repository includes multiple packages. The `api`
package contains core data types, interfaces and no-op implementations that comprise the OpenTelemetry API following
[the
specification](https://github.com/open-telemetry/opentelemetry-specification).
The `sdk` package is the reference implementation of the API.

Libraries that produce telemetry data should only depend on `api`
and defer the choice of the SDK to the application developer. Applications may
depend on `sdk` or another package that implements the API.

To install the API and SDK packages,

```
$ go get -u go.opentelemetry.io/otel
```

## Quick Start

```go
package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
	exporter, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func main() {
	initTracer()
	tracer := global.Tracer("ex.com/basic")

	tracer.WithSpan(context.Background(), "foo",
		func(ctx context.Context) error {
			tracer.WithSpan(ctx, "bar",
				func(ctx context.Context) error {
					tracer.WithSpan(ctx, "baz",
						func(ctx context.Context) error {
							return nil
						},
					)
					return nil
				},
			)
			return nil
		},
	)
}

```

See the [API
documentation](https://godoc.org/go.opentelemetry.io/otel) for more
detail, and the
[opentelemetry-example-app](./example/README.md)
for a complete example.

## Compatible Exporters

See the Go packages depending upon
[sdk/export/trace](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/export/trace?tab=importedby)
and [sdk/export/metric](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/export/metric?tab=importedby)
for a list of all exporters compatible with OpenTelemetry's Go SDK.

## Contributing

See the [contributing file](CONTRIBUTING.md).

## Release Schedule

OpenTelemetry Go is under active development. Below is the release schedule
for the Go library. The first version of the release isn't guaranteed to conform
to a specific version of the specification, and future releases will not
attempt to maintain backward compatibility with the alpha release.

| Component                        | Version      | Target Date      | Release Date     |
| -------------------------------- | ------------ | ---------------- | ---------------- |
| Tracing API                      | Alpha v0.1.0 | October 28 2019  | November 05 2019 |
| Tracing SDK                      | Alpha v0.1.0 | October 28 2019  | November 05 2019 |
| Jaeger Trace Exporter            | Alpha v0.1.0 | October 28 2019  | November 05 2019 |
| Trace Context Propagation        | Alpha v0.1.0 | Unknown          | November 05 2019 |
| OpenTracing Bridge               | Alpha v0.1.0 | October          | November 05 2019 |
| Metrics API                      | Alpha v0.2.0 | October 28 2019  | December 03 2019 |
| Metrics SDK                      | Alpha v0.2.0 | October 28 2019  | December 03 2019 |
| Prometheus Metrics Exporter      | Alpha v0.2.0 | October 28 2019  | December 03 2019 |
| Context Prop. rename/Baggage     | Alpha v0.3.0 | December 23 2019 | -                |
| OpenTelemetry Collector Exporter | Alpha v0.4.0 | January 15 2020  | -                |
| Zipkin Trace Exporter            | Alpha        | Unknown          | -                |
| OpenCensus Bridge                | Alpha        | Unknown          | -                |
