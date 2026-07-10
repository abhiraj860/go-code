package logging

import "fmt"

type PlainTextFormatter struct{}

func NewPlainTextFormatter() *PlainTextFormatter {
	return &PlainTextFormatter{}
}

func (formatter *PlainTextFormatter) Format(record LogRecord) string {
	return fmt.Sprintf(
		"%s [%s] [%s] %s",
		record.Timestamp().Format(timeLayout),
		record.Level().String(),
		record.ThreadName(),
		record.Message(),
	)
}
