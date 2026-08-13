package model

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"sync"
)

//go:generate go run ./internal/trimmodelinfo -in fhir-modelinfo-4.0.1.xml -out fhir-modelinfo-4.0.1.xml.gz

// modelInfoR4 is the ModelInfo cqframework publishes for FHIR 4.0.1, with the
// prose attributes stripped and gzipped by ./internal/trimmodelinfo. It is
// carried rather than transcribed for the same reason FHIRHelpers is: a
// hand-written subset is a divergence someone has to maintain, and the version
// it replaced described six resources out of 147.
//
//go:embed fhir-modelinfo-4.0.1.xml.gz
var modelInfoR4 []byte

var (
	r4Once sync.Once
	r4Info *StaticModelInfo
	r4Err  error
)

// FHIRModelInfo returns the model information for a FHIR version, or an error
// naming the versions that are available.
//
// Only 4.0.1 ships today: cqframework publishes no R5 ModelInfo, and the CQL
// IG's own Library/FHIR-ModelInfo is pinned to 4.0.1. Answering with an error
// rather than quietly serving R4 is the point — `using FHIR version '5.0.0'`
// used to be accepted in silence and evaluated against R4 anyway.
func FHIRModelInfo(version string) (ModelInfo, error) {
	switch version {
	case "4.0.1", "4.0.0", "":
		return LoadR4ModelInfo()
	default:
		return nil, fmt.Errorf("no model info for FHIR version %q; this build carries 4.0.1", version)
	}
}

// LoadR4ModelInfo parses the embedded R4 ModelInfo, once per process. Parsing
// 931 types is not free, and an engine is often built per request.
func LoadR4ModelInfo() (*StaticModelInfo, error) {
	r4Once.Do(func() {
		zr, err := gzip.NewReader(bytes.NewReader(modelInfoR4))
		if err != nil {
			r4Err = fmt.Errorf("opening embedded model info: %w", err)
			return
		}
		defer zr.Close()
		r4Info, r4Err = ParseModelInfo(zr)
	})
	return r4Info, r4Err
}
