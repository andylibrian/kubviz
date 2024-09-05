package otel

import (
	"context"
	stdlog "log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

const (
	logName = "github.com/intelops/kubviz/agent/kubviz/otel"
)

func PublishEventLog(eventName string, body []byte) {
	logRecord := log.Record{}
	logRecord.SetBody(log.BytesValue(body))
	logRecord.SetTimestamp(time.Now())
	logRecord.SetSeverity(log.SeverityTrace2)
	logRecord.AddAttributes(log.KeyValue{Key: "kubviz.event.name", Value: log.StringValue(eventName)})

	logger := global.GetLoggerProvider().Logger(logName)
	logger.Emit(context.Background(), logRecord)

	stdlog.Printf(">>> Debug: log record: %s\n", spew.Sdump(logRecord))
}
