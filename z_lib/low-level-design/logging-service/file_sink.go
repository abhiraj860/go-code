package logging

import "os"

type FileSink struct {
	file *os.File
}

func NewFileSink(filePath string) (*FileSink, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileSink{file: file}, nil
}

func (sink *FileSink) Write(formatted string) error {
	if _, err := sink.file.WriteString(formatted + "\n"); err != nil {
		return err
	}
	return sink.file.Sync()
}

func (sink *FileSink) Close() error {
	return sink.file.Close()
}
