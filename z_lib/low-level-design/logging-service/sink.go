package logging

type Sink interface {
	Write(formatted string) error
}
