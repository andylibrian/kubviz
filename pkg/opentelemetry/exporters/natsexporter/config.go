package natsexporter

// Config defines the configuration for NATS exporter.
type Config struct {
	Token      string `mapstructure:"token"`
	URL        string `mapstructure:"url"`
	StreamName string `mapstructure:"stream_name"`
}
