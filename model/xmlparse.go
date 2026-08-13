package model

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// The ELM ModelInfo schema, as published at urn:hl7-org:elm-modelinfo:r1. Only
// the parts the evaluator consults are modeled: type hierarchy and elements,
// which resolve properties and `is`/`as`; retrievable types and their primary
// code path, which a retrieve needs to know which element carries its codes;
// contexts and their key element, which is how a retrieve is narrowed to one
// patient; and the declared conversions between FHIR and system types.
type xmlModelInfo struct {
	Name                         string           `xml:"name,attr"`
	Version                      string           `xml:"version,attr"`
	PatientClassName             string           `xml:"patientClassName,attr"`
	PatientBirthDatePropertyName string           `xml:"patientBirthDatePropertyName,attr"`
	TypeInfos                    []xmlTypeInfo    `xml:"typeInfo"`
	ConversionInfos              []xmlConversion  `xml:"conversionInfo"`
	ContextInfos                 []xmlContextInfo `xml:"contextInfo"`
}

type xmlTypeInfo struct {
	Name            string                   `xml:"name,attr"`
	Namespace       string                   `xml:"namespace,attr"`
	BaseType        string                   `xml:"baseType,attr"`
	Retrievable     string                   `xml:"retrievable,attr"`
	PrimaryCodePath string                   `xml:"primaryCodePath,attr"`
	Elements        []xmlElement             `xml:"element"`
	ContextRels     []xmlContextRelationship `xml:"contextRelationship"`
}

// xmlContextRelationship names the search parameter that relates a type to a
// context. It is a search parameter and not an element path: Condition relates
// to Patient through "patient", and Condition has no element by that name.
type xmlContextRelationship struct {
	Context           string `xml:"context,attr"`
	RelatedKeyElement string `xml:"relatedKeyElement,attr"`
}

type xmlElement struct {
	Name          string       `xml:"name,attr"`
	ElementType   string       `xml:"elementType,attr"`
	TypeSpecifier *xmlTypeSpec `xml:"elementTypeSpecifier"`
}

type xmlTypeSpec struct {
	XSIType     string      `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	ElementType string      `xml:"elementType,attr"`
	Namespace   string      `xml:"namespace,attr"`
	Name        string      `xml:"name,attr"`
	Choices     []xmlChoice `xml:"choice"`
}

type xmlChoice struct {
	Namespace string `xml:"namespace,attr"`
	Name      string `xml:"name,attr"`
}

type xmlConversion struct {
	FunctionName string `xml:"functionName,attr"`
	FromType     string `xml:"fromType,attr"`
	ToType       string `xml:"toType,attr"`
}

type xmlContextInfo struct {
	Name             string      `xml:"name,attr"`
	KeyElement       string      `xml:"keyElement,attr"`
	BirthDateElement string      `xml:"birthDateElement,attr"`
	ContextType      xmlNamedRef `xml:"contextType"`
}

type xmlNamedRef struct {
	Namespace string `xml:"namespace,attr"`
	Name      string `xml:"name,attr"`
}

// ParseModelInfo reads an ELM ModelInfo document and builds the type metadata
// the evaluator consults.
func ParseModelInfo(r io.Reader) (*StaticModelInfo, error) {
	var doc xmlModelInfo
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing model info: %w", err)
	}
	if doc.Version == "" {
		return nil, fmt.Errorf("model info declares no version")
	}

	mi := NewStaticModelInfo(doc.Version)
	mi.patientClassName = doc.PatientClassName
	mi.patientBirthDatePath = doc.PatientBirthDatePropertyName

	for _, ti := range doc.TypeInfos {
		mi.addParsedType(ti)
	}

	for _, ci := range doc.ContextInfos {
		if ci.Name == "" {
			continue
		}
		resourceType := ci.ContextType.Name
		if resourceType == "" {
			resourceType = ci.Name
		}
		mi.AddContext(ci.Name, resourceType)
		if ci.KeyElement != "" {
			mi.contextKeys[ci.Name] = ci.KeyElement
		}
	}

	for _, cv := range doc.ConversionInfos {
		if cv.FromType == "" || cv.ToType == "" || cv.FunctionName == "" {
			continue
		}
		mi.conversions[conversionKey{From: cv.FromType, To: cv.ToType}] = cv.FunctionName
	}

	return mi, nil
}

// addParsedType registers one typeInfo entry.
func (m *StaticModelInfo) addParsedType(ti xmlTypeInfo) {
	if ti.Name == "" {
		return
	}
	info := &TypeInfo{
		Name:      ti.Name,
		Namespace: ti.Namespace,
		BaseName:  ti.BaseType,
		Elements:  make([]ElementInfo, 0, len(ti.Elements)),
	}
	// primaryCodePath only means anything for a type a retrieve can name.
	if ti.Retrievable == "true" {
		info.PrimaryKey = ti.PrimaryCodePath
		m.retrievable[ti.Name] = true
	}
	for _, rel := range ti.ContextRels {
		if rel.Context == "" || rel.RelatedKeyElement == "" {
			continue
		}
		key := contextRelKey{Type: ti.Name, Context: rel.Context}
		// A type may declare several relationships to one context; the first is
		// the one the model leads with.
		if _, seen := m.contextRels[key]; !seen {
			m.contextRels[key] = rel.RelatedKeyElement
		}
	}
	for _, el := range ti.Elements {
		if el.Name == "" {
			continue
		}
		info.Elements = append(info.Elements, elementInfoOf(el))
	}
	m.AddType(info)
}

// elementInfoOf renders one element, which the schema spells three ways: an
// elementType attribute, a nested list specifier, or a nested choice.
func elementInfoOf(el xmlElement) ElementInfo {
	out := ElementInfo{Name: el.Name, Type: el.ElementType}
	spec := el.TypeSpecifier
	if spec == nil {
		return out
	}
	switch {
	case strings.HasSuffix(spec.XSIType, "ListTypeSpecifier"):
		out.IsList = true
		out.Type = spec.ElementType
		// A list of choices spells its element type as a nested specifier
		// rather than an attribute.
		if out.Type == "" && spec.Name != "" {
			out.Type = qualify(spec.Namespace, spec.Name)
		}
	case strings.HasSuffix(spec.XSIType, "ChoiceTypeSpecifier"):
		out.IsChoice = true
		out.ChoiceTypes = make([]string, 0, len(spec.Choices))
		for _, c := range spec.Choices {
			if c.Name == "" {
				continue
			}
			out.ChoiceTypes = append(out.ChoiceTypes, qualify(c.Namespace, c.Name))
		}
	case strings.HasSuffix(spec.XSIType, "NamedTypeSpecifier"):
		out.Type = qualify(spec.Namespace, spec.Name)
	}
	return out
}

func qualify(namespace, name string) string {
	if namespace == "" || strings.Contains(name, ".") {
		return name
	}
	return namespace + "." + name
}
