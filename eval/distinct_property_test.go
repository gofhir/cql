package eval

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	cqltypes "github.com/gofhir/cql/types"
)

// quadraticDistinct is nullSafeDistinct as it was before it indexed: every
// element compared with every one kept before it. It is the oracle the
// bucketed version is held to, since the buckets are only allowed to save
// time, never to change an answer.
func quadraticDistinct(c fptypes.Collection) fptypes.Collection {
	if len(c) <= 1 {
		return c
	}
	result := make(fptypes.Collection, 0, len(c))
	for _, item := range c {
		if !nullSafeContains(result, item) {
			result = append(result, item)
		}
	}
	return result
}

// show writes a collection that may hold nulls, which Collection's own String
// does not survive.
func show(c fptypes.Collection) string {
	parts := make([]string, len(c))
	for i, v := range c {
		if v == nil {
			parts[i] = "null"
		} else {
			parts[i] = v.String()
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// TestBucketedDistinctAnswersAsTheQuadraticOne compares indexedDistinctPositions, called
// directly so that short lists reach it too, with the pairwise answer over random
// collections of values chosen to cross the buckets and the quantity index:
// numbers of both types and a dimensionless quantity that sameValue matches
// with them, quantities equal across units (mg and g, % and 1, calendar and
// UCUM) and a quantity with no unit, temporal values equal across type and
// offset, objects equal under another key order, objects that share an id and
// differ, and values with no bucket.
func TestBucketedDistinctAnswersAsTheQuadraticOne(t *testing.T) {
	must := func(v fptypes.Value, err error) fptypes.Value {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	quantity := func(s string) fptypes.Value {
		q, err := fptypes.NewQuantity(s)
		if err != nil {
			t.Fatal(err)
		}
		return q
	}
	object := func(s string) fptypes.Value { return fptypes.NewObjectValue([]byte(s)) }
	placed := func(s string, offset time.Duration) fptypes.Value {
		t.Helper()
		dt, err := fptypes.NewDateTime(s)
		if err != nil {
			t.Fatal(err)
		}
		return dt.WithDefaultOffset(offset)
	}

	pool := []fptypes.Value{
		nil,
		fptypes.NewInteger(1),
		must(fptypes.NewDecimal("1.0")),
		must(fptypes.NewDecimal("1.00")),
		fptypes.NewInteger(2),
		quantity("1 '1'"),
		quantity("1 'm'"),
		quantity("100 'cm'"),
		quantity("1000 'mg'"),
		quantity("1 'g'"),
		quantity("1 '%'"),
		quantity("100 '%'"),
		quantity("1"),
		quantity("100"),
		quantity("1 year"),
		quantity("1 'a'"),
		quantity("12 months"),
		fptypes.NewString("1"),
		fptypes.NewString("a"),
		fptypes.NewBoolean(true),
		fptypes.NewBoolean(false),
		must(fptypes.NewDate("2020-03-01")),
		must(fptypes.NewDateTime("2020-03-01")),
		must(fptypes.NewDateTime("2020-03-01T10:00:00Z")),
		must(fptypes.NewDateTime("2020-03-01T07:00:00-03:00")),
		must(fptypes.NewDate("2020")),
		must(fptypes.NewDateTime("2020")),
		must(fptypes.NewDate("2020-03")),
		must(fptypes.NewDateTime("2020-03")),
		must(fptypes.NewDateTime("2020-03-01T10:00:00")),
		placed("2020-03-01T10:00:00", -3*time.Hour),
		placed("2020-03-01T10:00:00", 0),
		must(fptypes.NewDateTime("2020-03-01T13:00:00Z")),
		must(fptypes.NewDateTime("2020-03-01T10:00:00.000Z")),
		must(fptypes.NewDateTime("2020-03-01T10")),
		placed("2020-03-01T10", -3*time.Hour),
		object(`{"start":"2020-01-01T00:00:00Z"}`),
		object(`{"start":"\u0032020-01-01T00:00:00Z"}`),
		object(`{"start":"2020-01-01T00:00:00+00:00"}`),
		object(`{"start":"2020-01-01T00:00:00Z","end":"2020-01-02T00:00:00Z"}`),
		object(`{"value":"\u0061"}`),
		object(`{"value":"a"}`),
		object(`{"resourceType":"Observation","id":"o1","status":"final"}`),
		object(`{"status":"final","id":"o1","resourceType":"Observation"}`),
		object(`{"resourceType":"Observation","id":"o1","status":"amended"}`),
		object(`{"resourceType":"Observation","id":"o2","status":"final"}`),
		object(`{"resourceType":"Observation","status":"final"}`),
		object(`{"status":"final","resourceType":"Observation"}`),
		object(`{"value":1,"unit":"mg"}`),
		object(`{"unit":"mg","value":1.0}`),
		object(`{"value":2,"unit":"mg"}`),
		object(`{"value":1,"unit":"g"}`),
		object(`{"value":"1","unit":"mg"}`),
		object(`{"value":null,"unit":"mg"}`),
		object(`{"unit":"mg"}`),
		object(`{"_value":{"extension":[{"url":"u"}]},"unit":"mg"}`),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": fptypes.NewInteger(1)}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": must(fptypes.NewDecimal("1.0"))}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": fptypes.NewInteger(1), "b": fptypes.NewString("x")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"b": fptypes.NewString("x"), "a": must(fptypes.NewDecimal("1.00"))}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": nil}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": quantity("1 'm'")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": quantity("1 '1'")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": quantity("100 '%'")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": quantity("1")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"a": quantity("100 'cm'")}),
		cqltypes.NewTuple(map[string]fptypes.Value{"t": cqltypes.NewTuple(map[string]fptypes.Value{"a": fptypes.NewInteger(1)})}),
		cqltypes.NewTuple(map[string]fptypes.Value{"t": cqltypes.NewTuple(map[string]fptypes.Value{"a": must(fptypes.NewDecimal("1.0"))})}),
		object(`{"coding":[{"system":"s","code":"c1"}]}`),
		object(`{"coding":[{"code":"c1","system":"s"}]}`),
		object(`{"coding":[{"system":"s","code":"c2"}]}`),
		object(`{"coding":[{"system":"s","code":"c1"},{"system":"s","code":"c2"}]}`),
		object(`{"coding":[{"system":"s","code":"\u0063\u0031"}]}`),
		object(`{"text":"x","coding":null}`),
		object(`{"text":"x"}`),
		object(`{"family":"F","_family":{"extension":[{"url":"u"}]}}`),
		object(`{"family":"F"}`),
		object(`{"given":["A"]}`),
		object(`{"given":"A"}`),
		cqltypes.NewList(fptypes.Collection{fptypes.NewInteger(1)}),
		cqltypes.NewInterval(fptypes.NewInteger(1), fptypes.NewInteger(2), true, true),
	}

	// except, intersect and union, each path against the pairwise answer:
	// every element of the first list tested against the whole second list.
	// A source of its own, so that the distinct loop below keeps the inputs
	// it has always had.
	setRNG := rand.New(rand.NewSource(2))
	for range 3000 {
		// Up to 60 a side, so that keepFirst takes its indexed branch too and
		// not only the pairwise one, which the oracle below is a copy of.
		lc := make(fptypes.Collection, setRNG.Intn(60))
		rc := make(fptypes.Collection, setRNG.Intn(60))
		for j := range lc {
			lc[j] = pool[setRNG.Intn(len(pool))]
		}
		for j := range rc {
			rc[j] = pool[setRNG.Intn(len(pool))]
		}
		for _, inRC := range []bool{false, true} {
			want := make(fptypes.Collection, 0, len(lc))
			for _, item := range lc {
				if nullSafeContains(rc, item) == inRC && !nullSafeContains(want, item) {
					want = append(want, item)
				}
			}
			for name, got := range map[string]fptypes.Collection{
				"indexed":   indexedKeepFirst(lc, rc, inRC, nil),
				"keepFirst": keepFirst(lc, rc, inRC, nil),
			} {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("keep of %s against %s (in=%t)\n  %s: %s\n  pairwise: %s",
						show(lc), show(rc), inRC, name, show(got), show(want))
				}
			}
		}
		union := make(fptypes.Collection, 0, len(lc)+len(rc))
		for _, item := range append(append(fptypes.Collection{}, lc...), rc...) {
			if !nullSafeContains(union, item) {
				union = append(union, item)
			}
		}
		for name, got := range map[string]fptypes.Collection{
			"indexed":       indexedUnion(lc, rc, make(fptypes.Collection, 0, len(lc)+len(rc)), nil),
			"nullSafeUnion": nullSafeUnion(lc, rc, nil),
		} {
			if !reflect.DeepEqual(got, union) {
				t.Fatalf("union of %s and %s\n  %s: %s\n  pairwise: %s", show(lc), show(rc), name, show(got), show(union))
			}
		}
	}
	rng := rand.New(rand.NewSource(1))
	for range 5000 {
		c := make(fptypes.Collection, rng.Intn(20))
		for j := range c {
			c[j] = pool[rng.Intn(len(pool))]
		}
		want := quadraticDistinct(c)
		// Both ways to the positions: indexed, and distinctPositions, which
		// compares a list this short pairwise. They have to keep the same
		// values, and the same first occurrence of each.
		for name, positions := range map[string][]int{
			"indexed":           indexedDistinctPositions(c, nil),
			"distinctPositions": distinctPositions(c, nil),
		} {
			got := make(fptypes.Collection, 0, len(c))
			for _, at := range positions {
				got = append(got, c[at])
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("distinct of %s\n  %s: %s\n  quadratic: %s", show(c), name, show(got), show(want))
			}
		}
	}
}
