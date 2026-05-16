package logging

type LogLevel int

const (
	DEBUG LogLevel = 10
	INFO  LogLevel = 20
	WARN  LogLevel = 30
	ERROR LogLevel = 40
	FATAL LogLevel = 50
)

func (level LogLevel) AtLeast(minimum LogLevel) bool {
	return level >= minimum
}

func (level LogLevel) String() string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
