package conformance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// parseExpectedOutput converts an expected output string from a CQL conformance
// test XML file into the corresponding fptypes.Value (or cqltypes value).
// Returns nil for "null". Returns an error for unsupported formats.
func parseExpectedOutput(raw string) (fptypes.Value, error) {
	s := strings.TrimSpace(raw)

	if s == "" || s == "null" {
		return nil, nil
	}

	// Boolean
	if s == "true" {
		return fptypes.NewBoolean(true), nil
	}
	if s == "false" {
		return fptypes.NewBoolean(false), nil
	}

	// Empty list
	if s == "{}" {
		return cqltypes.NewList(fptypes.Collection{}), nil
	}

	// List: {elem, elem, ...}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return parseList(s)
	}

	// Interval: Interval[2, 7] or Interval(2, 7]
	if strings.HasPrefix(s, "Interval") {
		return parseInterval(s)
	}

	// Tuple: Tuple { key: val, ... }
	if strings.HasPrefix(s, "Tuple") {
		return parseTuple(s)
	}

	// Concept: Concept { codes: Code { ... } ... }
	if strings.HasPrefix(s, "Concept") {
		return parseConcept(s)
	}

	// Code: Code { code: '...', system: '...', ... }
	if strings.HasPrefix(s, "Code") {
		return parseCode(s)
	}

	// Time: @T09:00:00.000
	if strings.HasPrefix(s, "@T") {
		timeStr := s[1:] // strip '@', keep 'T' prefix — NewTime accepts "T09:..."
		t, err := fptypes.NewTime(timeStr)
		if err != nil {
			return nil, fmt.Errorf("parsing time %q: %w", s, err)
		}
		return t, nil
	}

	// DateTime or Date starting with @
	if strings.HasPrefix(s, "@") {
		return parseDateTimeOrDate(s[1:]) // strip '@'
	}

	// Long literal (e.g., "1L", "3L") — parse as Integer (fits in int64)
	if longPattern.MatchString(s) {
		numStr := s[:len(s)-1] // strip trailing 'L'
		v, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing long %q: %w", s, err)
		}
		return fptypes.NewInteger(v), nil
	}

	// Quantity: number followed by single-quoted unit (e.g., 5.0'g', 19.99 '[lb_av]')
	if quantityPattern.MatchString(s) {
		return parseQuantity(s)
	}

	// String literal: 'hello'
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2 {
		inner := s[1 : len(s)-1]
		// Unescape \' → ' and \\ → \ and \" → "
		inner = strings.ReplaceAll(inner, "\\'", "'")
		inner = strings.ReplaceAll(inner, "\\\"", "\"")
		// Handle Unicode escape sequences \uXXXX
		inner = unescapeUnicode(inner)
		inner = strings.ReplaceAll(inner, "\\\\", "\\")
		return fptypes.NewString(inner), nil
	}

	// Integer (try before decimal to prefer integer for whole numbers like "42")
	if integerPattern.MatchString(s) {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing integer %q: %w", s, err)
		}
		return fptypes.NewInteger(v), nil
	}

	// Decimal
	if decimalPattern.MatchString(s) {
		d, err := fptypes.NewDecimal(s)
		if err != nil {
			return nil, fmt.Errorf("parsing decimal %q: %w", s, err)
		}
		return d, nil
	}

	return nil, fmt.Errorf("unrecognized output format: %q", s)
}

var unicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// unescapeUnicode replaces \uXXXX sequences with the corresponding Unicode character.
func unescapeUnicode(s string) string {
	return unicodeEscape.ReplaceAllStringFunc(s, func(match string) string {
		hex := match[2:] // strip \u
		code, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(code))
	})
}

var (
	integerPattern  = regexp.MustCompile(`^-?\d+$`)
	decimalPattern  = regexp.MustCompile(`^-?\d+\.\d+$`)
	longPattern     = regexp.MustCompile(`^-?\d+L$`)
	quantityPattern = regexp.MustCompile(`^-?[\d.]+\s*'[^']*'$`)
)

// parseDateTimeOrDate parses a string (already stripped of leading '@') as
// either a DateTime or Date. The rule: if the string contains 'T', it's a
// DateTime; otherwise it's a Date.
func parseDateTimeOrDate(s string) (fptypes.Value, error) {
	if strings.Contains(s, "T") {
		// DateTime. Strip trailing 'T' if no time component follows.
		// e.g., "2014T" -> "2014", "2014-01-01T" -> "2014-01-01"
		cleaned := strings.TrimSuffix(s, "T")
		// But only strip trailing T if it's truly a bare T (no digits after).
		// "2014-01-01T10:00:00" should stay as-is.
		// The regex: if the string ends with 'T' followed by nothing, strip it.
		if strings.HasSuffix(s, "T") && !strings.ContainsAny(s[strings.LastIndex(s, "T")+1:], "0123456789") {
			s = cleaned
		}
		dt, err := fptypes.NewDateTime(s)
		if err != nil {
			return nil, fmt.Errorf("parsing datetime %q: %w", s, err)
		}
		return dt, nil
	}

	// No T — it's a Date: "2014", "2014-01", "2014-01-01"
	d, err := fptypes.NewDate(s)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", s, err)
	}
	return d, nil
}

// parseQuantity parses a quantity string like `5.0'g'` or `19.99 '[lb_av]'`.
func parseQuantity(s string) (fptypes.Value, error) {
	// Ensure there's a space before the quoted unit for NewQuantity compatibility.
	// Find the position of the first single quote that starts the unit.
	idx := strings.Index(s, "'")
	if idx <= 0 {
		return nil, fmt.Errorf("invalid quantity format: %q", s)
	}
	numPart := strings.TrimSpace(s[:idx])
	unitPart := s[idx:] // includes quotes
	normalized := numPart + " " + unitPart

	q, err := fptypes.NewQuantity(normalized)
	if err != nil {
		return nil, fmt.Errorf("parsing quantity %q: %w", s, err)
	}
	return q, nil
}

// isBareKeyValue checks if a string looks like "Key: value" (identifier followed by colon).
var bareKeyValuePattern = regexp.MustCompile(`^\s*[A-Za-z_][A-Za-z0-9_]*\s*:`)

// looksLikeBareTuple checks if the inner content of braces looks like a bare tuple
// (all top-level elements are key: value pairs, not nested braces).
func looksLikeBareTuple(_ string, elements []string) bool {
	for _, elem := range elements {
		e := strings.TrimSpace(elem)
		if !bareKeyValuePattern.MatchString(e) {
			return false
		}
	}
	return true
}

// parseList parses a list literal like `{1, 2, 3}` or `{'a','b','c'}`.
// Also handles bare tuple syntax like `{ A: 2, B: 5 }`.
func parseList(s string) (fptypes.Value, error) {
	inner := strings.TrimSpace(s[1 : len(s)-1]) // strip { and }
	if inner == "" {
		return cqltypes.NewList(fptypes.Collection{}), nil
	}

	elements, err := splitTopLevel(inner)
	if err != nil {
		return nil, fmt.Errorf("splitting list elements: %w", err)
	}

	// Check if this is a bare tuple (all elements are key: value pairs)
	if looksLikeBareTuple(inner, elements) {
		return parseBareTuple(elements)
	}

	values := make(fptypes.Collection, 0, len(elements))
	for _, elem := range elements {
		v, err := parseExpectedOutput(strings.TrimSpace(elem))
		if err != nil {
			return nil, fmt.Errorf("parsing list element %q: %w", elem, err)
		}
		values = append(values, v)
	}
	return cqltypes.NewList(values), nil
}

// parseBareTuple parses a bare tuple from pre-split key:value elements.
func parseBareTuple(parts []string) (fptypes.Value, error) {
	elements := make(map[string]fptypes.Value, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid tuple element (no colon): %q", part)
		}
		key := strings.TrimSpace(part[:colonIdx])
		valStr := strings.TrimSpace(part[colonIdx+1:])
		val, err := parseExpectedOutput(valStr)
		if err != nil {
			return nil, fmt.Errorf("parsing tuple element %q: %w", key, err)
		}
		elements[key] = val
	}
	return cqltypes.NewTuple(elements), nil
}

// parseInterval parses an interval literal like `Interval[2, 7]` or `Interval(2, 7]`.
func parseInterval(s string) (fptypes.Value, error) {
	// Strip "Interval" prefix
	rest := strings.TrimPrefix(s, "Interval")
	rest = strings.TrimSpace(rest)

	if len(rest) < 3 {
		return nil, fmt.Errorf("invalid interval format: %q", s)
	}

	// Determine low/high closure from brackets
	lowClosed := rest[0] == '['
	if rest[0] != '[' && rest[0] != '(' {
		return nil, fmt.Errorf("invalid interval opening bracket in %q", s)
	}

	highClosed := rest[len(rest)-1] == ']'
	if rest[len(rest)-1] != ']' && rest[len(rest)-1] != ')' {
		return nil, fmt.Errorf("invalid interval closing bracket in %q", s)
	}

	// Extract the inner content (between brackets)
	inner := rest[1 : len(rest)-1]

	// Split on comma at the top level
	parts, err := splitTopLevel(inner)
	if err != nil {
		return nil, fmt.Errorf("splitting interval bounds: %w", err)
	}
	if len(parts) != 2 {
		return nil, fmt.Errorf("interval must have exactly 2 bounds, got %d in %q", len(parts), s)
	}

	low, err := parseExpectedOutput(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("parsing interval low bound: %w", err)
	}
	high, err := parseExpectedOutput(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("parsing interval high bound: %w", err)
	}

	return cqltypes.NewInterval(low, high, lowClosed, highClosed), nil
}

// parseTuple parses a tuple literal like `Tuple { id: 5, name: 'Chris'}`.
func parseTuple(s string) (fptypes.Value, error) {
	// Strip "Tuple" prefix and find the braces
	rest := strings.TrimPrefix(s, "Tuple")
	rest = strings.TrimSpace(rest)

	if !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
		return nil, fmt.Errorf("invalid tuple format: %q", s)
	}

	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	if inner == "" {
		return cqltypes.NewTuple(map[string]fptypes.Value{}), nil
	}

	parts, err := splitTopLevel(inner)
	if err != nil {
		return nil, fmt.Errorf("splitting tuple elements: %w", err)
	}

	elements := make(map[string]fptypes.Value, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid tuple element (no colon): %q", part)
		}
		key := strings.TrimSpace(part[:colonIdx])
		valStr := strings.TrimSpace(part[colonIdx+1:])
		val, err := parseExpectedOutput(valStr)
		if err != nil {
			return nil, fmt.Errorf("parsing tuple element %q: %w", key, err)
		}
		elements[key] = val
	}

	return cqltypes.NewTuple(elements), nil
}

// splitTopLevel splits a string by the given delimiter, but only at the top
// level — ignoring delimiters inside nested braces, brackets, parentheses,
// or single-quoted strings.
func splitTopLevel(s string) ([]string, error) {
	const delim byte = ','
	var parts []string
	depth := 0
	inQuote := false
	start := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			if ch == '\'' {
				inQuote = false
			}
			continue
		}
		switch ch {
		case '\'':
			inQuote = true
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
		case delim:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}

	if inQuote {
		return nil, fmt.Errorf("unterminated string literal in %q", s)
	}

	parts = append(parts, s[start:])
	return parts, nil
}

// parseConcept parses a Concept literal like:
//
//	Concept { codes: Code { code: '8480-6' } }
func parseConcept(s string) (fptypes.Value, error) {
	rest := strings.TrimPrefix(s, "Concept")
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
		return nil, fmt.Errorf("invalid Concept format: %q", s)
	}
	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	// Parse as key:value pairs using splitTopLevel
	parts, err := splitTopLevel(inner)
	if err != nil {
		return nil, fmt.Errorf("splitting Concept elements: %w", err)
	}
	var codes []cqltypes.Code
	display := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:colonIdx])
		valStr := strings.TrimSpace(part[colonIdx+1:])
		switch key {
		case "codes":
			// valStr could be a single Code or multiple; parse it
			codeVal, err := parseExpectedOutput(valStr)
			if err != nil {
				return nil, fmt.Errorf("parsing Concept codes: %w", err)
			}
			switch cv := codeVal.(type) {
			case cqltypes.Code:
				codes = append(codes, cv)
			case cqltypes.List:
				for _, item := range cv.Values {
					if c, ok := item.(cqltypes.Code); ok {
						codes = append(codes, c)
					}
				}
			}
		case "display":
			if strings.HasPrefix(valStr, "'") && strings.HasSuffix(valStr, "'") {
				display = valStr[1 : len(valStr)-1]
			}
		}
	}
	return cqltypes.NewConcept(codes, display), nil
}

// parseCode parses a Code literal like:
//
//	Code { code: '8480-6', system: 'http://loinc.org', display: 'Systolic BP' }
func parseCode(s string) (fptypes.Value, error) {
	rest := strings.TrimPrefix(s, "Code")
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
		return nil, fmt.Errorf("invalid Code format: %q", s)
	}
	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	parts, err := splitTopLevel(inner)
	if err != nil {
		return nil, fmt.Errorf("splitting Code elements: %w", err)
	}
	code := ""
	system := ""
	display := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:colonIdx])
		valStr := strings.TrimSpace(part[colonIdx+1:])
		if strings.HasPrefix(valStr, "'") && strings.HasSuffix(valStr, "'") {
			valStr = valStr[1 : len(valStr)-1]
		}
		switch key {
		case "code":
			code = valStr
		case "system":
			system = valStr
		case "display":
			display = valStr
		}
	}
	return cqltypes.NewCode(system, code, display), nil
}
