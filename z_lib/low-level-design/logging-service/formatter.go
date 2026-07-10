package logging

type Formatter interface {
	Format(record LogRecord) string
}
