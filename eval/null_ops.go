package eval

import (
	fptypes "github.com/gofhir/fhirpath/types"
)

// IsNull returns true if the value is null.
func IsNull(v fptypes.Value) fptypes.Value {
	return fptypes.NewBoolean(v == nil)
}

// IsTrue returns true if the value is true (not null and not false).
func IsTrue(v fptypes.Value) fptypes.Value {
	if v == nil {
		return fptypes.NewBoolean(false)
	}
	b, ok := v.(fptypes.Boolean)
	if !ok {
		return fptypes.NewBoolean(false)
	}
	return fptypes.NewBoolean(b.Bool())
}

// IsFalse returns true if the value is false (not null and not true).
func IsFalse(v fptypes.Value) fptypes.Value {
	if v == nil {
		return fptypes.NewBoolean(false)
	}
	b, ok := v.(fptypes.Boolean)
	if !ok {
		return fptypes.NewBoolean(false)
	}
	return fptypes.NewBoolean(!b.Bool())
}

// Coalesce returns the first non-null value from the arguments.
func Coalesce(values ...fptypes.Value) fptypes.Value {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// IfNull returns the first value if non-null, otherwise the second.
func IfNull(a, b fptypes.Value) fptypes.Value {
	if a != nil {
		return a
	}
	return b
}

// ThreeValuedAnd implements CQL three-valued AND logic:
//
//	true AND true → true
//	true AND false → false
//	true AND null → null
//	false AND true → false
//	false AND false → false
//	false AND null → false
//	null AND true → null
//	null AND false → false
//	null AND null → null
func ThreeValuedAnd(left, right fptypes.Value) fptypes.Value {
	l := toBoolOrNil(left)
	r := toBoolOrNil(right)

	if l != nil && r != nil {
		return fptypes.NewBoolean(*l && *r)
	}
	if l != nil && !*l {
		return fptypes.NewBoolean(false)
	}
	if r != nil && !*r {
		return fptypes.NewBoolean(false)
	}
	return nil // null
}

// ThreeValuedOr implements CQL three-valued OR logic:
//
//	true OR true → true
//	true OR false → true
//	true OR null → true
//	false OR true → true
//	false OR false → false
//	false OR null → null
//	null OR true → true
//	null OR false → null
//	null OR null → null
func ThreeValuedOr(left, right fptypes.Value) fptypes.Value {
	l := toBoolOrNil(left)
	r := toBoolOrNil(right)

	if l != nil && r != nil {
		return fptypes.NewBoolean(*l || *r)
	}
	if l != nil && *l {
		return fptypes.NewBoolean(true)
	}
	if r != nil && *r {
		return fptypes.NewBoolean(true)
	}
	return nil // null
}

// ThreeValuedNot implements CQL three-valued NOT logic:
//
//	NOT true → false
//	NOT false → true
//	NOT null → null
func ThreeValuedNot(v fptypes.Value) fptypes.Value {
	b := toBoolOrNil(v)
	if b == nil {
		return nil
	}
	return fptypes.NewBoolean(!*b)
}

// ThreeValuedImplies implements CQL implies logic:
//
//	true implies true → true
//	true implies false → false
//	true implies null → null
//	false implies anything → true
//	null implies true → true
//	null implies false/null → null
func ThreeValuedImplies(left, right fptypes.Value) fptypes.Value {
	l := toBoolOrNil(left)
	r := toBoolOrNil(right)

	if l != nil && !*l {
		return fptypes.NewBoolean(true)
	}
	if r != nil && *r {
		return fptypes.NewBoolean(true)
	}
	if l != nil && *l && r != nil {
		return fptypes.NewBoolean(*r)
	}
	return nil
}

func toBoolOrNil(v fptypes.Value) *bool {
	if v == nil {
		return nil
	}
	b, ok := v.(fptypes.Boolean)
	if !ok {
		return nil
	}
	val := b.Bool()
	return &val
}
