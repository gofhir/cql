package conformance

import "encoding/xml"

// TestSuite is the root <tests> element.
type TestSuite struct {
	XMLName      xml.Name     `xml:"tests"`
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Groups       []TestGroup  `xml:"group"`
	Capabilities []Capability `xml:"capability"`
}

// TestGroup is a <group> element containing related tests.
type TestGroup struct {
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Tests        []TestCase   `xml:"test"`
	Capabilities []Capability `xml:"capability"`
}

// TestCase is a single <test> element.
type TestCase struct {
	Name         string       `xml:"name,attr"`
	Version      string       `xml:"version,attr"`
	Ordered      string       `xml:"ordered,attr"`
	Expression   Expression   `xml:"expression"`
	Outputs      []Output     `xml:"output"`
	Capabilities []Capability `xml:"capability"`
}

// Expression is the <expression> element containing CQL to evaluate.
type Expression struct {
	Invalid string `xml:"invalid,attr"`
	Value   string `xml:",chardata"`
}

// Output is an <output> element with the expected result.
type Output struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// Capability declares a required capability for a test.
type Capability struct {
	Code  string `xml:"code,attr"`
	Value string `xml:"value,attr"`
}
