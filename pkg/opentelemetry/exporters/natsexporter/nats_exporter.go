package natsexporter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type natsExporter struct {
	conf           *Config
	subjectName    string
	logger         *zap.Logger
	logsMarshaller LogsMarshaler
	jetStream      *JetStream
}

func (n *natsExporter) consumeLogs(ctx context.Context, logs plog.Logs) error {
	marshalled, err := n.logsMarshaller.Marshal(logs)
	if err != nil {
		return err
	}

	err = n.jetStream.Publish(n.subjectName, marshalled)
	if err != nil {
		return err
	}

	// TODO: remove
	fmt.Printf("natsExporter.consumeLogs() Log Marshalled: %s\n", string(marshalled))

	return nil
}

func (n *natsExporter) Start(_ context.Context, host component.Host) error {
	n.logger.Info("natsexporter: started")
	return nil
}

func (n *natsExporter) Shutdown(context.Context) error {
	n.logger.Info("natsexporter: shutting down")
	return nil
}
