package otel

import (
	"context"
	stdlog "log"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	logsdk "go.opentelemetry.io/otel/sdk/log"
)

const (
	logName         = "github.com/intelops/kubviz/agent/kubviz/otel"
	kubvizEventName = "kubviz.event.name"
	logTimeout      = 3 * time.Second
)

var loggerProvider *logsdk.LoggerProvider

func newLoggerProvider() (*logsdk.LoggerProvider, error) {
	// Use env var to configure the endpoint:
	// OTEL_EXPORTER_OTLP_ENDPOINT=grpc://localhost:4317
	exporter, err := otlploggrpc.New(context.Background())

	if err != nil {
		return nil, err
	}

	loggerProvider := logsdk.NewLoggerProvider(
		logsdk.WithProcessor(logsdk.NewBatchProcessor(exporter)),
	)

	return loggerProvider, nil
}

func init() {
	var err error
	loggerProvider, err = newLoggerProvider()
	if err != nil {
		stdlog.Fatalln(err.Error())
	}
}

func PublishEventLog(eventName string, body []byte) {
	logRecord := log.Record{}
	logRecord.SetBody(log.StringValue(string(body)))
	logRecord.SetTimestamp(time.Now())
	logRecord.SetSeverity(log.SeverityTrace2)
	logRecord.AddAttributes(log.KeyValue{Key: kubvizEventName, Value: log.StringValue(eventName)})

	ctx, cancel := context.WithTimeout(context.Background(), logTimeout)
	defer cancel()

	logger := loggerProvider.Logger(logName)
	logger.Emit(ctx, logRecord)
}
