package cql

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// referenceEncoderCases are the answers the reference harness has to write for
// testdata/reference/encoder.cql, one definition per branch of its encoder
// (internal/referenceharness/src/Evaluate.java, encode), written by hand: each
// is the value its CQL denotes (CQL 1.5.3, Types and Literals; the clinical
// values, Code, Concept, Tuple and Interval), encoded as the harness README
// says. They do not depend on the reference's version, and hold the encoder
// when the harness moves to another one (plan 007).
var referenceEncoderCases = map[string]string{
	"D0": `{"type": "Boolean", "value": true}`,
	"D1": `{"type": "Integer", "value": "2147483647"}`,
	"D2": `{"type": "Long", "value": "9007199254740993"}`,
	"D3": `{"type": "Decimal", "value": "1.50"}`,
	"D4": `{"type": "String", "value": "a \"b\""}`,
	"D5": `null`,
	"D6": `{"type": "Quantity", "value": "5.5", "unit": "mg"}`,
	"D7": `{"type": "DateTime", "value": "2020-01-02T03:04:05.006+02:00", "precision": "millisecond", "offset": "+02:00", "instant": "2020-01-02T03:04:05.006+02:00"}`,
	// A DateTime with no offset takes the evaluation's, -03:00 (Evaluate.java,
	// EVALUATION); its instant is the start of its precision.
	"D8":  `{"type": "DateTime", "value": "2020-01", "precision": "month", "offset": "-03:00", "instant": "2020-01-01T00:00:00-03:00"}`,
	"D9":  `{"type": "Date", "value": "2020-01-02", "precision": "day"}`,
	"D10": `{"type": "Time", "value": "10:11", "precision": "minute"}`,
	"D11": `{"type": "Interval", "low": {"type": "Integer", "value": "1"}, "lowClosed": true, "high": {"type": "Integer", "value": "5"}, "highClosed": false}`,
	"D12": `{"type": "Interval", "low": null, "lowClosed": false, "high": {"type": "Date", "value": "2020-01-02", "precision": "day"}, "highClosed": true}`,
	"D13": `{"type": "List", "elements": [{"type": "Integer", "value": "1"}, null, {"type": "Integer", "value": "3"}]}`,
	"D14": `{"type": "List", "elements": []}`,
	"D15": `{"type": "Code", "code": "c", "system": "s", "version": "v", "display": "d"}`,
	// Every field of a Code is written, null included.
	"D16": `{"type": "Code", "code": "c", "system": null, "version": null, "display": null}`,
	"D17": `{"type": "Concept", "codes": [{"type": "Code", "code": "c", "system": "s", "version": null, "display": null}], "display": "x"}`,
	"D18": `{"type": "Tuple", "fields": {"a": {"type": "Integer", "value": "1"}, "b": {"type": "String", "value": "x"}, "c": null}}`,
	"D19": `{"type": "Tuple", "fields": {"l": {"type": "List", "elements": [{"type": "Tuple", "fields": {"x": {"type": "Integer", "value": "1"}}}]}}}`,
	// A FHIR primitive: its model type and the text of its value.
	"D20": `{"type": "FHIR.string", "value": "a"}`,
	"D21": `{"type": "FHIR.date", "value": "1980-01-01"}`,
	// A FHIR element: its model type and its FHIR JSON (FHIR R4, JSON
	// Representation), its extension included.
	"D22": `{"type": "FHIR.Coding", "value": {"extension": [{"url": "http://example.org/e", "valueString": "x"}], "system": "s", "code": "c"}}`,
	// The resource is the one fhir-data.json serves, as the test reads it.
	"D23": "Patient",
	"D24": `{"type": "List", "elements": [{"type": "FHIR.date", "value": "1980-01-01"}]}`,
}

// TestTheReferenceEncoderWritesEachValueAsItIs holds what the Java harness
// wrote for encoder.cql to the values written by hand above. A FHIR value's
// JSON is compared as JSON, since the order of its keys is the serializer's.
// Regenerate it, from internal/referenceharness, with encoder.cql,
// fhir-data.json and FHIRHelpers-4.0.1.cql copied to <dir>:
//
//	docker run --rm -v "$PWD":/w -v <dir>:/cql -w /w maven:3-eclipse-temurin-21 \
//	  java -cp "out:lib/*" Evaluate -each /cql/encoder.cql /cql /cql/fhir-data.json > encoder.java.json
func TestTheReferenceEncoderWritesEachValueAsItIs(t *testing.T) {
	dir := filepath.Join("testdata", "reference")
	src, err := os.ReadFile(filepath.Join(dir, "encoder.cql"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "encoder.java.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		SourceSha256 string                     `json:"sourceSha256"`
		Engine       string                     `json:"engine"`
		Results      map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(src)
	if ref.SourceSha256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("encoder.java.json is of another encoder.cql: %s", ref.SourceSha256)
	}
	if want := pinnedReferenceEngine(t, "java"); ref.Engine != want {
		t.Errorf("encoder.java.json was run with %q; the harness pins %q", ref.Engine, want)
	}
	defines := snapshotDefine.FindAllStringSubmatch(string(src), -1)
	if len(defines) != len(referenceEncoderCases) || len(ref.Results) != len(referenceEncoderCases) {
		t.Fatalf("encoder.cql has %d definitions and the reference %d answers; the cases are %d",
			len(defines), len(ref.Results), len(referenceEncoderCases))
	}
	for name, want := range referenceEncoderCases {
		got, ok := ref.Results[name]
		if !ok {
			t.Errorf("%s: the reference wrote no answer", name)
			continue
		}
		if want == "Patient" {
			want = referenceEncoderPatient(t)
		}
		if g, w := encodedValue(t, got), encodedValue(t, json.RawMessage(want)); !reflect.DeepEqual(g, w) {
			t.Errorf("%s: the harness wrote %s; want %s", name, got, want)
		}
	}
}

// referenceEncoderPatient is the encoding the Patient fhir-data.json serves
// has to have.
func referenceEncoderPatient(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "snapshots", "fhir-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Patient json.RawMessage `json:"Patient"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(map[string]any{"type": "FHIR.Patient", "value": data.Patient})
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// encodedValue decodes an encoded answer, and the FHIR JSON a FHIR value holds
// as text into the JSON it is, so that two serializations of one resource
// compare equal.
func encodedValue(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	if m, ok := v.(map[string]any); ok {
		if text, isText := m["value"].(string); isText && text != "" && text[0] == '{' {
			var inner any
			if err := json.Unmarshal([]byte(text), &inner); err != nil {
				t.Fatalf("%s: %v", text, err)
			}
			m["value"] = inner
		}
	}
	return v
}
