package logging

import "fmt"

type ConsoleSink struct{}

func NewConsoleSink() *ConsoleSink {
	return &ConsoleSink{}
}

func (sink *ConsoleSink) Write(formatted string) error {
	fmt.Println(formatted)
	return nil
}
