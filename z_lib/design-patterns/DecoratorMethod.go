// Structural Patterns
// Decorator: Use when you need to layer optional behaviors at runtime without subclass explosion.

// NOTE: This is the Decorator pattern (object composition),
// not Go's function literals or middleware chains, though the idea is similar.

package main

import "fmt"

type DataSource interface {
	WriteData(data string)
	ReadData() string
}

type FileDataSource struct {
	filename string
	storage string // simple in-memory storage for example
}

func NewFileDataSource(filename string) *FileDataSource {
	return &FileDataSource{filename: filename}
}

func (f *FileDataSource) WriteData(data string) {
	f.storage = data
}

func (f *FileDataSource) ReadData() string {
	return f.storage
}

type EncryptionDecorator struct {
	wrapped DataSource
}

func NewEncryptionDecorator(source DataSource) *EncryptionDecorator {
	return &EncryptionDecorator{wrapped : source}
}

func (d *EncryptionDecorator) WriteData(data string) {
	encrypted := "encrypted: " + data
	d.wrapped.WriteData(encrypted)
}

func (d *EncryptionDecorator) ReadData() string {
	data := d.wrapped.ReadData()
	return trimPrefix(data, "encrypted:")
}

type CompressionDecorator struct {
	wrapped DataSource
}

func NewCompressionDecorator(source DataSource) *CompressionDecorator {
	return &CompressionDecorator{wrapped : source}
}

func (d *CompressionDecorator) WriteData(data string) {
	compressed := "compressed: " + data
	d.wrapped.WriteData(compressed) 
} 

func (d *CompressionDecorator) ReadData() string {
	data := d.wrapped.ReadData()
	return trimPrefix(data, "compressed:")
}

func trimPrefix(data, prefix string) string {
	if len(data) >= len(prefix) && data[:len(prefix)] == prefix {
		return data[len(prefix):]
	}
	return data
}

func main() {
	// Usage:
	var source DataSource = NewFileDataSource("data.txt")
	source = NewEncryptionDecorator(source)
	source = NewCompressionDecorator(source)
	source.WriteData("sensitive info")
	fmt.Println(source.ReadData())
}

















