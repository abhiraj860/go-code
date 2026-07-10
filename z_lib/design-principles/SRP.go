// SOLID Principles
// SRP - Keep Classes focussed on one responsibility

package main

type Report struct {}

type PDFPrinter struct{}

func (PDFPrinter) Printer(report Report) {
	_ = report
	// PDF formatting
}

type FileStorage struct{}

func (FileStorage) Save(content string) {
	_ = content
	// file I/O
}