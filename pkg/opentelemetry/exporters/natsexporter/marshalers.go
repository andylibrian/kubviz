package natsexporter

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

type LogsMarshaler interface {
	Marshal(logs plog.Logs) ([]byte, error)
}

type pdataLogsMarshaler struct {
	marshaler plog.Marshaler
}

func createLogMarshaler() LogsMarshaler {
	return newPdataLogsMarshaler(&plog.JSONMarshaler{})
}

func newPdataLogsMarshaler(marshaler plog.Marshaler) LogsMarshaler {
	return pdataLogsMarshaler{
		marshaler: marshaler,
	}
}

func (p pdataLogsMarshaler) Marshal(ld plog.Logs) ([]byte, error) {
	bts, err := p.marshaler.MarshalLogs(ld)

	return bts, err
}
