package logging

import "time"

const timeLayout = time.RFC3339Nano

type Logger struct {
	destinations []*Destination
}

func NewLogger(destinations []*Destination) *Logger {
	copied := make([]*Destination, len(destinations))
	copy(copied, destinations)
	return &Logger{destinations: copied}
}

func (logger *Logger) Log(level LogLevel, message string) {
	record := NewLogRecord(
		time.Now().UTC(),
		level,
		message,
		"goroutine",
	)

	for _, destination := range logger.destinations {
		destination.Write(record)
	}
}

func (logger *Logger) Debug(message string) {
	logger.Log(DEBUG, message)
}

func (logger *Logger) Info(message string) {
	logger.Log(INFO, message)
}

func (logger *Logger) Warn(message string) {
	logger.Log(WARN, message)
}

func (logger *Logger) Error(message string) {
	logger.Log(ERROR, message)
}

func (logger *Logger) Fatal(message string) {
	logger.Log(FATAL, message)
}

