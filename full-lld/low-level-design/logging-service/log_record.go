package logging

import "time"

type LogRecord struct {
	timestamp  time.Time
	level      LogLevel
	message    string
	threadName string
}

func NewLogRecord(timestamp time.Time, level LogLevel, message string, threadName string) LogRecord {
	return LogRecord{
		timestamp:  timestamp,
		level:      level,
		message:    message,
		threadName: threadName,
	}
}

func (record LogRecord) Timestamp() time.Time {
	return record.timestamp
}

func (record LogRecord) Level() LogLevel {
	return record.level
}

func (record LogRecord) Message() string {
	return record.message
}

func (record LogRecord) ThreadName() string {
	return record.threadName
}
