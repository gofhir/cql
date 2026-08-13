// Command trimmodelinfo produces the compressed ModelInfo the model package
// embeds, from the document cqframework publishes.
//
// The published file is 3.4 MB, most of it prose: every element carries a
// description and a definition meant for humans reading the spec, and the
// evaluator reads none of it. Dropping those and compressing leaves 100 KB,
// which is small enough to embed and keeps the file recognizable as the
// upstream document rather than a translation of it.
//
// Usage:
//
//	go run ./model/internal/trimmodelinfo -in fhir-modelinfo-4.0.1.xml -out model/fhir-modelinfo-4.0.1.xml.gz
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"regexp"
)

// Attributes that exist to be read by people, not by the evaluator.
var prose = regexp.MustCompile(`\s+(description|definition|comment|label)="(?:[^"\\]|\\.)*"`)

var indent = regexp.MustCompile(`\n\s*`)

func main() {
	in := flag.String("in", "", "path to the published fhir-modelinfo XML")
	out := flag.String("out", "", "path to write the trimmed, gzipped document to")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "both -in and -out are required")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", *in, err)
		os.Exit(1)
	}
	trimmed := indent.ReplaceAll(prose.ReplaceAll(raw, nil), []byte("\n"))

	if err := writeCompressed(*out, trimmed); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d KB → %d KB\n", *out, len(raw)/1024, fileSize(*out)/1024)
}

// writeCompressed writes the trimmed document, closing everything in order so
// that a failure to flush is reported rather than producing a truncated file.
func writeCompressed(path string, content []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	zw, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		f.Close()
		return fmt.Errorf("compressing: %w", err)
	}
	if _, err := zw.Write(content); err != nil {
		zw.Close()
		f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return fmt.Errorf("flushing %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}

func fileSize(path string) int {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return int(fi.Size())
}
