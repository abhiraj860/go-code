package logging

import "encoding/json"

type JsonFormatter struct{}

type jsonLogRecord struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Thread    string `json:"thread"`
	Message   string `json:"message"`
}

func NewJsonFormatter() *JsonFormatter {
	return &JsonFormatter{}
}

func (formatter *JsonFormatter) Format(record LogRecord) string {
	value := jsonLogRecord{
		Timestamp: record.Timestamp().Format(timeLayout),
		Level:     record.Level().String(),
		Thread:    record.ThreadName(),
		Message:   record.Message(),
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}

	return string(encoded)
}

