package logging

import (
	"fmt"
	"os"
	"sync"
)

type Destination struct {
	formatter Formatter
	minLevel  LogLevel
	sink      Sink
	mutex     sync.Mutex
}

func NewDestination(formatter Formatter, minLevel LogLevel, sink Sink) *Destination {
	return &Destination{
		formatter: formatter,
		minLevel:  minLevel,
		sink:      sink,
	}
}

func (destination *Destination) Write(record LogRecord) {
	if !record.Level().AtLeast(destination.minLevel) {
		return
	}

	formatted := destination.formatter.Format(record)

	destination.mutex.Lock()
	defer destination.mutex.Unlock()

	if err := destination.sink.Write(formatted); err != nil {
		fmt.Fprintln(os.Stderr, "logger: sink write failed: "+err.Error())
	}
}
