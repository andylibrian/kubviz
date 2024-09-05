package natsexporter

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/intelops/kubviz/pkg/opentelemetry/exporters/natsexporter/internal/metadata"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

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
		// exporter.WithTraces(createTracesExporter, metadata.TracesStability),
		// exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability))
}

type NatsExporter interface {
	component.Component
	// consumeTraces(_ context.Context, td ptrace.Traces) error
	// consumeMetrics(_ context.Context, md pmetric.Metrics) error
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
	fe := getOrCreateFileExporter(cfg, set.Logger)
	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		fe.consumeLogs,
		exporterhelper.WithStart(fe.Start),
		exporterhelper.WithShutdown(fe.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

func getOrCreateFileExporter(cfg component.Config, logger *zap.Logger) NatsExporter {
	conf := cfg.(*Config)
	return newFileExporter(conf, logger)
}

func newFileExporter(conf *Config, logger *zap.Logger) NatsExporter {
	logsMarshaller, err := createLogMarshaler()
	if err != nil {
		logger.Fatal("error wile creating logs Marshaller")
	}

	level := logrus.InfoLevel

	logger2 := logrus.New()
	logger2.SetLevel(level)

	logger2.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
	})

	logger2.SetOutput(os.Stdout)

	streamName := "otel-debug"
	js, err := NewJetstream(logger2, "nats://localhost:4222", []nats.Option{}, streamName)
	if err != nil {
		log.Fatal(err)
	}

	err = js.Connect()
	if err != nil {
		log.Fatal(err)
	}

	streamConfig := nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamName},
		Retention: nats.LimitsPolicy,
		Discard:   nats.DiscardOld,
		Storage:   nats.FileStorage,
	}

	err = js.Init(streamConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &natsExporter{
		conf:           conf,
		logger:         logger,
		logsMarshaller: logsMarshaller,
		jetStream:      js,
	}
}

type natsExporter struct {
	conf           *Config
	logger         *zap.Logger
	logsMarshaller LogsMarshaler
	jetStream      *JetStream
}

// LogsMarshaler marshals logs into Message array
type LogsMarshaler interface {
	Marshal(logs plog.Logs, topic string) ([]byte, error)
}

type pdataLogsMarshaler struct {
	marshaler plog.Marshaler
}

// creates LogsMarshalers based on the provided config
func createLogMarshaler() (LogsMarshaler, error) {
	return newPdataLogsMarshaler(&plog.JSONMarshaler{}), nil
}

func newPdataLogsMarshaler(marshaler plog.Marshaler) LogsMarshaler {
	return pdataLogsMarshaler{
		marshaler: marshaler,
	}
}

func (p pdataLogsMarshaler) Marshal(ld plog.Logs, topic string) ([]byte, error) {
	bts, err := p.marshaler.MarshalLogs(ld)

	return bts, err
}

func (n *natsExporter) consumeLogs(ctx context.Context, ld plog.Logs) error {
	marshalled, err := n.logsMarshaller.Marshal(ld, "")
	if err != nil {
		return err
	}

	// test unmarshall
	// unmarshaller := &plog.JSONUnmarshaler{}
	// ld2, err := unmarshaller.UnmarshalLogs(marshalled)
	// if err != nil {
	// 	return err
	// }

	streamName := "otel-debug"
	err = n.jetStream.publishWithRetry(streamName, marshalled)
	if err != nil {
		return err
	}

	fmt.Printf("natsExporter.consumeLogs() Log Marshalled: %s\n", string(marshalled))

	return nil
}

func (n *natsExporter) Start(_ context.Context, host component.Host) error {
	fmt.Println("natsExporter.Start()")
	return nil
}

func (n *natsExporter) Shutdown(context.Context) error {
	fmt.Println("natsExporter.Shutdown()")
	return nil
}
