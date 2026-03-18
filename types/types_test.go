package types

import (
	"testing"

	fptypes "github.com/gofhir/fhirpath/types"
)

func TestInterval_Contains(t *testing.T) {
	iv := NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)

	tests := []struct {
		name     string
		value    fptypes.Value
		expected bool
	}{
		{"within range", fptypes.NewInteger(5), true},
		{"at low boundary", fptypes.NewInteger(1), true},
		{"at high boundary", fptypes.NewInteger(10), true},
		{"below range", fptypes.NewInteger(0), false},
		{"above range", fptypes.NewInteger(11), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := iv.Contains(tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Contains(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestInterval_OpenBoundaries(t *testing.T) {
	iv := NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), false, false)

	// Open boundaries should exclude endpoints
	result, err := iv.Contains(fptypes.NewInteger(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("open low boundary should exclude 1")
	}

	result, err = iv.Contains(fptypes.NewInteger(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("open high boundary should exclude 10")
	}

	result, err = iv.Contains(fptypes.NewInteger(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("5 should be within (1, 10)")
	}
}

func TestInterval_Equal(t *testing.T) {
	a := NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	b := NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(10), true, true)
	c := NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(5), true, true)

	if !a.Equal(b) {
		t.Error("equal intervals should be equal")
	}
	if a.Equal(c) {
		t.Error("different intervals should not be equal")
	}
}

func TestTuple_GetAndEqual(t *testing.T) {
	a := NewTuple(map[string]fptypes.Value{
		"name": fptypes.NewString("test"),
		"age":  fptypes.NewInteger(30),
	})

	v, ok := a.Get("name")
	if !ok || v.String() != "test" {
		t.Errorf("expected name=test, got %v", v)
	}

	v, ok = a.Get("missing")
	if ok {
		t.Errorf("expected missing key to return false")
	}
	if v != nil {
		t.Errorf("expected nil for missing key")
	}
}

func TestCode_Equal(t *testing.T) {
	a := NewCode("http://loinc.org", "12345-6", "Test")
	b := NewCode("http://loinc.org", "12345-6", "Test")
	c := NewCode("http://loinc.org", "99999-9", "Other")

	if !a.Equal(b) {
		t.Error("equal codes should be equal")
	}
	if a.Equal(c) {
		t.Error("different codes should not be equal")
	}
}

func TestList_Operations(t *testing.T) {
	list := NewList(fptypes.Collection{
		fptypes.NewInteger(1),
		fptypes.NewInteger(2),
		fptypes.NewInteger(3),
	})

	if list.IsEmpty() {
		t.Error("list should not be empty")
	}
	if len(list.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(list.Values))
	}

	empty := NewList(nil)
	if !empty.IsEmpty() {
		t.Error("empty list should be empty")
	}
}

func TestConcept_Equal(t *testing.T) {
	a := NewConcept([]Code{
		NewCode("http://example.org", "A", "Code A"),
	}, "Concept A")
	b := NewConcept([]Code{
		NewCode("http://example.org", "A", "Code A"),
	}, "Concept A")
	c := NewConcept([]Code{
		NewCode("http://example.org", "B", "Code B"),
	}, "Concept B")

	if !a.Equal(b) {
		t.Error("equal concepts should be equal")
	}
	if a.Equal(c) {
		t.Error("different concepts should not be equal")
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — CodeRef, CodeSystemRef, ValueSetRef, Ratio
// ---------------------------------------------------------------------------

func TestCodeRef(t *testing.T) {
	cr := NewCodeRef("MyCode", "")
	if cr.Type() != "CodeRef" {
		t.Errorf("type = %s, want CodeRef", cr.Type())
	}
	if cr.IsEmpty() {
		t.Error("CodeRef should not be empty")
	}
	if cr.String() != "CodeRef(MyCode)" {
		t.Errorf("string = %s, want CodeRef(MyCode)", cr.String())
	}

	// With library qualifier
	cr2 := NewCodeRef("MyCode", "Lib1")
	if cr2.String() != "CodeRef(Lib1.MyCode)" {
		t.Errorf("string = %s, want CodeRef(Lib1.MyCode)", cr2.String())
	}

	// Equal always returns false (deferred ref)
	if cr.Equal(cr) { //nolint:gocritic // intentional self-equality test
		t.Error("CodeRef.Equal should always be false")
	}
}

func TestCodeSystemRef(t *testing.T) {
	csr := NewCodeSystemRef("LOINC", "")
	if csr.Type() != "CodeSystemRef" {
		t.Errorf("type = %s, want CodeSystemRef", csr.Type())
	}
	if csr.String() != "CodeSystemRef(LOINC)" {
		t.Errorf("string = %s, want CodeSystemRef(LOINC)", csr.String())
	}

	csr2 := NewCodeSystemRef("LOINC", "Lib1")
	if csr2.String() != "CodeSystemRef(Lib1.LOINC)" {
		t.Errorf("string = %s, want CodeSystemRef(Lib1.LOINC)", csr2.String())
	}
}

func TestValueSetRef(t *testing.T) {
	vsr := NewValueSetRef("Diabetes", "")
	if vsr.Type() != "ValueSetRef" {
		t.Errorf("type = %s, want ValueSetRef", vsr.Type())
	}
	if vsr.String() != "ValueSetRef(Diabetes)" {
		t.Errorf("string = %s, want ValueSetRef(Diabetes)", vsr.String())
	}

	vsr2 := NewValueSetRef("Diabetes", "Lib1")
	if vsr2.String() != "ValueSetRef(Lib1.Diabetes)" {
		t.Errorf("string = %s, want ValueSetRef(Lib1.Diabetes)", vsr2.String())
	}

	if vsr.Equal(vsr) { //nolint:gocritic // intentional self-equality test
		t.Error("ValueSetRef.Equal should always be false")
	}
}

func TestRatio(t *testing.T) {
	r := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(128))
	if r.Type() != "Ratio" {
		t.Errorf("type = %s, want Ratio", r.Type())
	}
	if r.IsEmpty() {
		t.Error("Ratio should not be empty")
	}
	if r.String() != "1 : 128" {
		t.Errorf("string = %q, want %q", r.String(), "1 : 128")
	}
}

func TestRatio_Equal(t *testing.T) {
	a := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(2))
	b := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(2))
	c := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(3))

	if !a.Equal(b) {
		t.Error("equal ratios should be equal")
	}
	if a.Equal(c) {
		t.Error("different ratios should not be equal")
	}
	// Different type
	if a.Equal(fptypes.NewInteger(1)) {
		t.Error("ratio should not equal integer")
	}
}

func TestRatio_Equivalent(t *testing.T) {
	a := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(2))
	b := NewRatio(fptypes.NewInteger(1), fptypes.NewInteger(2))

	if !a.Equivalent(b) {
		t.Error("equivalent ratios should be equivalent")
	}
	if a.Equivalent(fptypes.NewString("x")) {
		t.Error("ratio should not be equivalent to string")
	}
}

func TestRatio_NullComponents(t *testing.T) {
	r := NewRatio(nil, fptypes.NewInteger(2))
	if r.String() != "null : 2" {
		t.Errorf("string = %q, want %q", r.String(), "null : 2")
	}
}
