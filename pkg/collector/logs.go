package collector

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type StateWriter interface {
	Write(ctx context.Context, collectorName, name, namespace, path string, writer func(w io.Writer) error) error
}

type LogCollector struct{}

func NewLogCollector() *LogCollector {
	return &LogCollector{}
}

func (l *LogCollector) Name() string {
	return "Logs"
}

func (l *LogCollector) Collect(ctx context.Context, name, namespace string, writer StateWriter) error {
	reader := strings.NewReader("EXAMPLE")

	err := writer.Write(ctx, l.Name(), name, namespace, fmt.Sprintf("%s/example", l.Name()), func(w io.Writer) error {
		_, writeErr := io.Copy(w, reader)
		return writeErr
	})

	return err
}
