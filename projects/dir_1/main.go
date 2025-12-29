package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Option func(*CliConfig) error

type CliConfig struct {
	OutputFile string
	ErrStream  io.Writer
	OutStream  io.Writer
}

func NewCliConfig(opts ...Option) (CliConfig, error) {
	c := CliConfig{
		OutputFile: "",
		ErrStream:  os.Stderr,
		OutStream:  os.Stdout,
	}

	for _, op := range opts {
		if err := op(&c); err != nil {
			return CliConfig{}, err
		}
	}

	return c, nil
}

func UpdateErrStream(inpErrStream io.Writer) Option {
	return func(c *CliConfig) error {
		c.ErrStream = inpErrStream
		return nil
	}
}

func UpdateOutStream(outStream io.Writer) Option {
	return func(c *CliConfig) error {
		c.OutStream = outStream
		return nil
	}
}

func app(directories []string, outputWriter io.Writer, cfg *CliConfig) {
	for _, directory := range directories {
		err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
			if path == ".git" {
				return filepath.SkipDir
			}
			if d.IsDir() {
				fmt.Fprintf(outputWriter, "%s\n", path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(cfg.ErrStream, "Error in walking the path %q: %v\n", directory, err)
			continue
		}
	}
}

func main() {
	var outputFile string
	var outputWriter io.Writer

	flag.StringVar(&outputFile, "f", "", "Output file (default:stdout)")
	flag.Parse()

	directories := os.Args[1:]

	if len(outputFile) != 0 {
		directories = os.Args[3:]
	}

	if len(directories) == 0 {
		fmt.Fprintf(os.Stderr, "No words provided")
		os.Exit(1)
	}
	cfg, err := NewCliConfig()
	cfg.OutputFile = outputFile
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
	}
	if cfg.OutputFile != "" {
		outputFile, err := os.Create(cfg.OutputFile)
		if err != nil {
			fmt.Fprintf(cfg.ErrStream, "Error Creating output file: %v\n", err)
			os.Exit(1)
		}
		defer outputFile.Close()
		outputWriter = io.MultiWriter(cfg.OutStream, outputFile)
	} else {
		outputWriter = cfg.OutStream
	}
	app(directories, outputWriter, &cfg)
}
