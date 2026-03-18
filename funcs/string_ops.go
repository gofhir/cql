package funcs

import (
	"regexp"
	"strings"
	"sync"

	fptypes "github.com/gofhir/fhirpath/types"
)

// regexCache caches compiled regular expressions to avoid repeated compilation.
var regexCache sync.Map // string → *regexp.Regexp

// getOrCompileRegex returns a cached compiled regex or compiles and caches it.
func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}

// Combine joins a collection of strings with a separator.
func Combine(c fptypes.Collection, separator string) fptypes.Value {
	parts := make([]string, 0, c.Count())
	for _, item := range c {
		if item != nil {
			parts = append(parts, item.String())
		}
	}
	return fptypes.NewString(strings.Join(parts, separator))
}

// Split splits a string by the given separator.
func Split(s fptypes.Value, separator string) fptypes.Value {
	if s == nil {
		return nil
	}
	sv, ok := s.(fptypes.String)
	if !ok {
		return nil
	}
	parts := strings.Split(sv.Value(), separator)
	result := make(fptypes.Collection, 0, len(parts))
	for _, p := range parts {
		result = append(result, fptypes.NewString(p))
	}
	return collectionToList(result)
}

// Length returns the length of a string.
func Length(s fptypes.Value) fptypes.Value {
	if s == nil {
		return fptypes.NewInteger(0)
	}
	if sv, ok := s.(fptypes.String); ok {
		return fptypes.NewInteger(int64(len(sv.Value())))
	}
	return fptypes.NewInteger(0)
}

// Upper converts a string to uppercase.
func Upper(s fptypes.Value) fptypes.Value {
	if s == nil {
		return nil
	}
	if sv, ok := s.(fptypes.String); ok {
		return fptypes.NewString(strings.ToUpper(sv.Value()))
	}
	return s
}

// Lower converts a string to lowercase.
func Lower(s fptypes.Value) fptypes.Value {
	if s == nil {
		return nil
	}
	if sv, ok := s.(fptypes.String); ok {
		return fptypes.NewString(strings.ToLower(sv.Value()))
	}
	return s
}

// StartsWith checks if a string starts with a prefix.
func StartsWith(s, prefix fptypes.Value) fptypes.Value {
	if s == nil || prefix == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	pv, ok2 := prefix.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewBoolean(false)
	}
	return fptypes.NewBoolean(strings.HasPrefix(sv.Value(), pv.Value()))
}

// EndsWith checks if a string ends with a suffix.
func EndsWith(s, suffix fptypes.Value) fptypes.Value {
	if s == nil || suffix == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	ev, ok2 := suffix.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewBoolean(false)
	}
	return fptypes.NewBoolean(strings.HasSuffix(sv.Value(), ev.Value()))
}

// Substring extracts a substring.
func Substring(s fptypes.Value, start, length int) fptypes.Value {
	if s == nil {
		return nil
	}
	sv, ok := s.(fptypes.String)
	if !ok {
		return nil
	}
	str := sv.Value()
	if start < 0 || start >= len(str) {
		return fptypes.NewString("")
	}
	end := start + length
	if length <= 0 || end > len(str) {
		end = len(str)
	}
	return fptypes.NewString(str[start:end])
}

// IndexOf returns the index of a substring, or -1 if not found.
func IndexOf(s, substring fptypes.Value) fptypes.Value {
	if s == nil || substring == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	sub, ok2 := substring.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewInteger(-1)
	}
	return fptypes.NewInteger(int64(strings.Index(sv.Value(), sub.Value())))
}

// Matches checks if a string matches a regex pattern (full match).
func Matches(s, pattern fptypes.Value) fptypes.Value {
	if s == nil || pattern == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	pv, ok2 := pattern.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewBoolean(false)
	}
	re, err := getOrCompileRegex("^(?:" + pv.Value() + ")$")
	if err != nil {
		return fptypes.NewBoolean(false)
	}
	return fptypes.NewBoolean(re.MatchString(sv.Value()))
}

// ReplaceMatches replaces regex matches in a string.
func ReplaceMatches(s, pattern, replacement fptypes.Value) fptypes.Value {
	if s == nil || pattern == nil || replacement == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	pv, ok2 := pattern.(fptypes.String)
	rv, ok3 := replacement.(fptypes.String)
	if !ok1 || !ok2 || !ok3 {
		return s
	}
	re, err := getOrCompileRegex(pv.Value())
	if err != nil {
		return s
	}
	return fptypes.NewString(re.ReplaceAllString(sv.Value(), rv.Value()))
}

// PositionOf returns the 0-based index of the first occurrence of pattern in string.
func PositionOf(pattern, s fptypes.Value) fptypes.Value {
	if s == nil || pattern == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	pv, ok2 := pattern.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewInteger(-1)
	}
	return fptypes.NewInteger(int64(strings.Index(sv.Value(), pv.Value())))
}

// LastPositionOf returns the 0-based index of the last occurrence of pattern in string.
func LastPositionOf(pattern, s fptypes.Value) fptypes.Value {
	if s == nil || pattern == nil {
		return nil
	}
	sv, ok1 := s.(fptypes.String)
	pv, ok2 := pattern.(fptypes.String)
	if !ok1 || !ok2 {
		return fptypes.NewInteger(-1)
	}
	return fptypes.NewInteger(int64(strings.LastIndex(sv.Value(), pv.Value())))
}

// collectionToList wraps a collection in a CQL List via a simple interface.
// We use a lightweight wrapper to avoid importing cqltypes (avoid circular dep).
type listValue struct {
	items fptypes.Collection
}

func (l listValue) Type() string                    { return "List" }
func (l listValue) Equal(o fptypes.Value) bool      { return false }
func (l listValue) Equivalent(o fptypes.Value) bool { return false }
func (l listValue) String() string                  { return "List" }
func (l listValue) IsEmpty() bool                   { return len(l.items) == 0 }

func collectionToList(c fptypes.Collection) fptypes.Value {
	return listValue{items: c}
}
