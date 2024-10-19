package natsexporter

import (
	"context"

	"github.com/intelops/kubviz/pkg/opentelemetry/exporters/natsexporter/internal/metadata"
	"github.com/nats-io/nats.go"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability))
}

type NatsExporter interface {
	component.Component
	consumeLogs(_ context.Context, ld plog.Logs) error
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	natsExporter := getOrCreateNatsExporter(cfg, set.Logger)
	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		natsExporter.consumeLogs,
		exporterhelper.WithStart(natsExporter.Start),
		exporterhelper.WithShutdown(natsExporter.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func getOrCreateNatsExporter(cfg component.Config, logger *zap.Logger) NatsExporter {
	conf := cfg.(*Config)
	return newNatsExporter(conf, logger)
}

func newNatsExporter(conf *Config, logger *zap.Logger) NatsExporter {
	logsMarshaler := createLogMarshaler()

	streamName := conf.StreamName
	natsOptions := []nats.Option{}
	if conf.Token != "" {
		natsOptions = append(natsOptions, nats.Token(conf.Token))
	}

	js := NewJetstream(logger, conf.URL, natsOptions, streamName)

	err := js.Connect()
	if err != nil {
		logger.Error("failed to connect to NATS server", zap.Error(err))
	}

	streamConfig := nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{conf.SubjectName},
		Retention: nats.LimitsPolicy,
		Discard:   nats.DiscardOld,
		Storage:   nats.FileStorage,
	}

	err = js.Init(streamConfig)
	if err != nil {
		logger.Error("failed to init NATS Jetstream", zap.Error(err))
	}

	return &natsExporter{
		conf:           conf,
		logger:         logger,
		logsMarshaller: logsMarshaler,
		jetStream:      js,
		subjectName:    conf.SubjectName,
	}
}
