package cql

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"

	"github.com/gofhir/cql/eval"
)

// Plan 018 (docs/plans/018-date-to-datetime.md): a DateTime coarser than the
// hour is not placed at the evaluation offset for comparison, and extracting its
// timezone offset answers the evaluation request's offset all the same.

// obsProvider serves an Observation whose effective[x] is a FHIR dateTime
// written to the day, and an Encounter whose period starts with a time and no
// offset.
type obsProvider struct{}

func (obsProvider) Retrieve(_ context.Context, r eval.RetrieveRequest) ([]json.RawMessage, error) {
	switch r.ResourceType {
	case "Encounter":
		return []json.RawMessage{json.RawMessage(
			`{"resourceType":"Encounter","id":"e1","status":"finished",` +
				`"period":{"start":"2020-06-15T23:00:00","end":"2020-06-16T23:00:00"}}`)}, nil
	case "Observation":
	default:
		return nil, nil
	}
	return []json.RawMessage{json.RawMessage(
		`{"resourceType":"Observation","id":"o1","status":"final","code":{"text":"x"},` +
			`"effectiveDateTime":"2020-03-01"}`)}, nil
}

// evaluationOffsets are the offsets the request is made at, in seconds, among
// them a fractional one.
var evaluationOffsets = []int{0, -5 * 3600, 5*3600 + 30*60, 14 * 3600, -11 * 3600}

// evalValueAt evaluates expr with the evaluation request at the given offset.
func evalValueAt(t *testing.T, seconds int, expr string) fptypes.Value {
	t.Helper()
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\n" +
		"include FHIRHelpers version '4.0.1' called FH\ncontext Patient\ndefine A: " + expr + "\n"
	got, err := NewEngine(WithDataProvider(obsProvider{}), WithEvaluationTimestamp(
		time.Date(2020, 6, 1, 12, 0, 0, 0, time.FixedZone("", seconds)))).
		EvaluateExpression(context.Background(), src, "A",
			[]byte(`{"resourceType":"Patient","id":"p1"}`), nil)
	if err != nil {
		t.Fatalf("%s at %+ds: %v", expr, seconds, err)
	}
	return got
}

func hoursOf(seconds int) string {
	return strconv.FormatFloat(float64(seconds)/3600, 'f', -1, 64)
}

func TestTimezoneOffsetOfAnUnplacedDateTimeIsTheEvaluationOffset(t *testing.T) {
	for _, expr := range []string{
		"timezoneoffset from @2020T",
		"timezoneoffset from @2020-03T",
		"timezoneoffset from @2020-03-01T",
		"timezoneoffset from @2020-03-01T10",
		"timezoneoffset from @2020-03-01T10:00:00.000",
		"timezoneoffset from ToDateTime(@2020-03-01)",
		"timezoneoffset from DateTime(2020)",
		"timezoneoffset from DateTime(2020, 3, 1)",
		"timezoneoffset from (@2020-03-01T + 1 day)",
		"timezoneoffset from (First([Observation]).effective as FHIR.dateTime)",
		"timezoneoffset from FH.ToDateTime(First([Observation]).effective as FHIR.dateTime)",
	} {
		for _, secs := range evaluationOffsets {
			got := evalValueAt(t, secs, expr)
			if got == nil || got.String() != hoursOf(secs) {
				t.Errorf("%s at %+ds = %v, want %s", expr, secs, got, hoursOf(secs))
			}
		}
	}
}

func TestTimezoneOffsetKeepsWhatTheValueStates(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"timezoneoffset from @2020-03-01T10:00:00.000+07:00", "7"},
		{"timezoneoffset from @2020-03-01T10:00:00.000-03:30", "-3.5"},
		{"timezoneoffset from @2020-03-01T10:00:00.000Z", "0"},
		{"timezoneoffset from @2020-03-01T10+05:30", "5.5"},
		{"timezoneoffset from minimum DateTime", "0"},
		{"timezoneoffset from maximum DateTime", "0"},
	} {
		for _, secs := range evaluationOffsets {
			got := evalValueAt(t, secs, tt.expr)
			if got == nil || got.String() != tt.want {
				t.Errorf("%s at %+ds = %v, want %s", tt.expr, secs, got, tt.want)
			}
		}
	}
	for _, secs := range evaluationOffsets {
		if got := evalValueAt(t, secs, "timezoneoffset from (null as DateTime)"); got != nil {
			t.Errorf("timezoneoffset from null at %+ds = %v, want null", secs, got)
		}
	}
}

// A mixed list of a day-precision value and a value with an offset keeps main's
// answers (round 1, finding 2 of the plan): the day value is not placed, so it
// is the earlier of the two and the +14:00 value the later.
func TestMinMaxSortOfADayAndAnOffsetValueStayAsTheyWere(t *testing.T) {
	const list = "{ @2020-03-01T, @2020-03-01T10:00:00.000+14:00 }"
	for _, tt := range []struct{ expr, want string }{
		{"Min(" + list + ") = @2020-03-01T", "true"},
		{"Max(" + list + ") = @2020-03-01T10:00:00.000+14:00", "true"},
		{"First((" + list + ") L sort asc) = @2020-03-01T", "true"},
		{"Last((" + list + ") L sort asc) = @2020-03-01T10:00:00.000+14:00", "true"},
		{"First((" + list + ") L sort desc) = @2020-03-01T10:00:00.000+14:00", "true"},
	} {
		got := evalValueAt(t, -3*3600, tt.expr)
		if got == nil || got.String() != tt.want {
			t.Errorf("%s at -03:00 = %v, want %s", tt.expr, got, tt.want)
		}
	}
}

// offsetOf reports the offset a DateTime answer is placed at.
func offsetOf(t *testing.T, v fptypes.Value) (minutes int, placed bool) {
	t.Helper()
	dt, ok := v.(fptypes.DateTime)
	if !ok {
		t.Fatalf("answer is %T (%v), want a DateTime", v, v)
	}
	return dt.EffectiveOffset()
}

// Each of the five places that put a DateTime at the evaluation offset places
// one specified to the hour or finer and leaves a coarser one unplaced, at a
// whole and at a fractional offset. The answers are read through EffectiveOffset,
// which is what a comparison reads.
func TestEachPlaceOfDefaultOffsetPlacesFromTheHour(t *testing.T) {
	for _, tt := range []struct {
		site     string
		expr     string
		wantPlac bool
	}{
		{"literal, hour", "@2020-03-01T10", true},
		{"literal, millisecond", "@2020-03-01T10:00:00.000", true},
		{"literal, day", "@2020-03-01T", false},
		{"literal, month", "@2020-03T", false},
		{"situate, a FHIR dateTime with a time and no offset", "First([Encounter]).period.start", true},
		{"situate, a FHIR dateTime written to the day", "First([Observation]).effective as FHIR.dateTime", false},
		{"situate, ToDateTime of a Date", "ToDateTime(@2020-03-01)", false},
		{"funcs carryDefaultOffset, the high boundary of an hour value", "HighBoundary(@2020-03-01T10, 17)", true},
		{"shifted value (carryDefaultOffsetTo or the arithmetic), hour", "@2020-03-01T10 + 1 hour", true},
		{"shifted value, day", "@2020-03-01T + 1 day", false},
	} {
		for _, secs := range evaluationOffsets {
			got := evalValueAt(t, secs, tt.expr)
			if got == nil {
				t.Errorf("%s: %s at %+ds is null", tt.site, tt.expr, secs)
				continue
			}
			minutes, placed := offsetOf(t, got)
			if placed != tt.wantPlac || (placed && minutes != secs/60) {
				t.Errorf("%s: %s at %+ds is at %d minutes, placed %v; want placed %v at %d",
					tt.site, tt.expr, secs, minutes, placed, tt.wantPlac, secs/60)
			}
		}
	}
	// The point a timing phrase moves keeps the frame of its source, so the
	// phrase answers the same as the comparison of the two points.
	for _, secs := range evaluationOffsets {
		for _, tt := range []struct{ expr, want string }{
			{"@2020-03-01T10:30:00.000 within 1 hour of @2020-03-01T10:00:00.000", "true"},
			{"@2020-03-01T12:30:00.000 within 1 hour of @2020-03-01T10:00:00.000", "false"},
		} {
			if got := evalValueAt(t, secs, tt.expr); got == nil || got.String() != tt.want {
				t.Errorf("%s at %+ds = %v, want %s", tt.expr, secs, got, tt.want)
			}
		}
	}
}

// Known limit (plan 018, round 2, finding 2): a caller that places a DateTime
// coarser than the hour itself, with fhirpath's own WithDefaultOffset, gets a
// value the engine's guard never sees. fhirpath v1.16.7 compares it as placed
// and the literal beside it as not, so the comparison cannot be decided. With
// fhirpath v1.16.6 this call did nothing. gofhir/server, the only consumer, does
// not make this call; README.md tells callers not to.
func TestACallerThatPlacesADayDateTimeItselfIsAKnownLimit(t *testing.T) {
	day, err := fptypes.NewDateTime("2020-03-01")
	if err != nil {
		t.Fatal(err)
	}
	placed := day.WithDefaultOffset(-3 * time.Hour)
	if got := askParamAtOffset(t, -3, placed, "P = @2020-03-01T"); got != "null" {
		t.Errorf("a day DateTime placed by the caller = @2020-03-01T answers %s; the documented limit is null", got)
	}
	// What the documentation asks for: the value as it is, which compares.
	if got := askParamAtOffset(t, -3, day, "P = @2020-03-01T"); got != "true" {
		t.Errorf("a day DateTime left unplaced = @2020-03-01T answers %s, want true", got)
	}
}

// An accepted change of fhirpath v1.16.7 (plan 018, round 2, finding 1): a
// decimal out of the range of v1.16.6 is read, so an extension that holds it is
// present.
func TestADecimalOutOfRangeInAResourceIsNowRead(t *testing.T) {
	src := "library T version '1.0'\nusing FHIR version '4.0.1'\ncontext Patient\n" +
		"define A: exists(Patient.extension)\n"
	got, err := NewEngine().EvaluateExpression(context.Background(), src, "A",
		[]byte(`{"resourceType":"Patient","id":"p1","extension":[{"url":"http://x","valueDecimal":1e1001}]}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.String() != "true" {
		t.Errorf("exists(Patient.extension) with valueDecimal 1e1001 = %v, want true", got)
	}
}

// Only types.WithDefaultOffset may call fhirpath's WithDefaultOffset, so that no
// place of the engine can put a DateTime coarser than the hour at an offset
// again. An area that answers the same either way would not notice a call that
// went back to the library.
func TestOnlyTheGuardCallsWithDefaultOffset(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata", "referenceharness":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.ToSlash(path) == "types/default_offset.go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		// Every selection of the method, not only a direct call: a method value
		// (`place := dt.WithDefaultOffset`) reaches the library just the same.
		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "WithDefaultOffset" {
				// types.WithDefaultOffset, the guard itself, is a package selection.
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "cqltypes" {
					return true
				}
				t.Errorf("%s selects %s.WithDefaultOffset; call types.WithDefaultOffset instead",
					fset.Position(sel.Pos()), exprName(sel.X))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func exprName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return exprName(x.X) + "." + x.Sel.Name
	}
	return "value"
}
