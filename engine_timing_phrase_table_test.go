package cql

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// evalTimingPhrase evaluates at a fixed offset, so a phrase measured in months
// does not depend on where the test runs.
func evalTimingPhrase(t *testing.T, expr string) string {
	t.Helper()
	got, err := NewEngine(WithEvaluationTimestamp(
		time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC))).
		EvaluateExpression(context.Background(),
			"library T version '1.0'\ndefine A: "+expr+"\n", "A", nil, nil)
	if err != nil {
		// An expression this table cannot evaluate is a defect in the table, not a
		// result to compare against another. Returning the error as a string let
		// two spellings agree on being broken: `1 day less than before` is not CQL,
		// both sides of the pair failed to parse, and the invariant passed on the
		// strength of two identical error messages. A comparison that cannot
		// compare has to fail.
		t.Errorf("could not evaluate %q: %v", expr, err)
		return "«unevaluated: " + expr + "»"
	}
	if got == nil {
		return "null"
	}
	return got.String()
}

// TestEverySpellingOfATimingPhraseAgrees crosses the dimensions a timing phrase
// has and asserts that spellings CQL makes synonymous answer alike.
//
// This exists because covering them one case at a time did not converge. The
// quantity offset and the boundary word were both dropped at parse time, and
// fixing them took three review rounds and fifteen findings — each one another
// corner of the same space: the word read for one phrase kind and not another, for
// a point operand and not an interval, from Low/High in one path and Start/End in
// the other. The space is
//
//	starts | ends | occurs | (nothing)
//	  ×  before | after | on or before | on or after
//	  ×  a point or an interval on the right
//	  ×  with or without a quantity offset
//	  ×  open or closed bounds, and Date as well as DateTime
//
// and the way to hold it is to state the property rather than the answers: what a
// phrase means is not in question here, only that two ways of writing one thing
// cannot disagree. That is the same instrument this repository reached for in
// v1.15.2 (interval equality through seven paths) and v1.20.x (four readers of one
// temporal frame), both times after the case-by-case approach had missed some.
//
// It found `within` unimplemented on its first run: the phrase was parsed, its
// quantity dropped, and evaluated as plain `during`. That is fixed, and the
// phrase is a row of the table now rather than a note beside it.
func TestEverySpellingOfATimingPhraseAgrees(t *testing.T) {
	lefts := map[string]string{
		"closed bounds": "Interval[@2019-03-01T00:00:00, @2019-09-01T00:00:00]",
		"open bounds":   "Interval(@2019-03-01T00:00:00, @2019-09-01T00:00:00)",
		"dates":         "Interval[@2019-03-01, @2019-09-01]",
		// An interval whose start and whose whole give different answers about
		// the rights below. Without one, the equivalence between naming an end
		// and taking that end first holds for the wrong reason — every left here
		// answered the same either way, and `occurs within` read as a boundary
		// word passed 936 pairs while taking the start of an interval the bare
		// spelling asked about whole.
		"a start inside and an end outside": "Interval[@2019-06-01T00:00:00, @2019-12-01T00:00:00]",
	}
	rights := map[string]string{
		"a point inside": "@2019-06-01T00:00:00",
		"a point after":  "@2019-11-01T00:00:00",
		"an interval":    "Interval[@2019-01-01T00:00:00, @2019-12-31T00:00:00]",
		"a date":         "@2019-06-01",
	}
	relationships := []string{"before", "after", "on or before", "on or after"}
	// `within Q of` carries its own quantity inside the relation, so it crosses the
	// table on its own rather than against the offset prefixes.
	withins := []string{"within 1 month of", "within 6 months of"}
	// The strict qualifier comes before the quantity and the inclusive one after
	// it, so each is written the way its own rule spells it.
	offsets := []string{
		"", "1 month or less ", "1 month or more ",
		"less than 1 month ", "more than 1 month ", "6 months or less ",
	}

	pairs := 0
	// The two equivalences, in one place so a relation with a different shape is
	// held to exactly the same ones.
	agree := func(leftName, rightName, left, right, off, rel string) {
		t.Helper()
		// The boundary word names an end, and naming it is the same as taking that
		// end first.
		for _, w := range []struct{ word, extractor string }{
			{"starts", "start of"},
			{"ends", "end of"},
		} {
			phrase := fmt.Sprintf("%s %s %s%s %s", left, w.word, off, rel, right)
			extracted := fmt.Sprintf("%s %s %s%s %s", w.extractor, left, off, rel, right)
			pairs++
			if a, b := evalTimingPhrase(t, phrase), evalTimingPhrase(t, extracted); a != b {
				t.Errorf("[%s, %s] `%s` = %s but `%s` = %s — one name for one end",
					leftName, rightName, phrase, a, extracted, b)
			}
		}
		// `occurs` is the default written out, so writing it changes nothing.
		phrase := fmt.Sprintf("%s occurs %s%s %s", left, off, rel, right)
		bare := fmt.Sprintf("%s %s%s %s", left, off, rel, right)
		pairs++
		if a, b := evalTimingPhrase(t, phrase), evalTimingPhrase(t, bare); a != b {
			t.Errorf("[%s, %s] `%s` = %s but `%s` = %s — occurs is the default said aloud",
				leftName, rightName, phrase, a, bare, b)
		}
	}

	for leftName, left := range lefts {
		for rightName, right := range rights {
			// `within` is here rather than in a test of its own because it is the
			// same claim: two spellings of one phrase cannot disagree. It used to,
			// and in the worst way — `starts within 2 months of` said false for a
			// point one month away while the extracted spelling said null.
			for _, rel := range withins {
				agree(leftName, rightName, left, right, "", rel)
			}
			for _, rel := range relationships {
				for _, off := range offsets {
					agree(leftName, rightName, left, right, off, rel)
				}
			}
		}
	}
	// The count is asserted exactly, not as a floor: a floor set below the real
	// figure would not notice a whole row of the table being dropped, which is how
	// the stale "720" survived a sixth offset being added.
	// lefts × rights × (relationships × offsets + withins) × pairs per cell
	const expected = 4 * 4 * (4*6 + 2) * 3
	if pairs != expected {
		t.Errorf("compared %d pairs, expected %d — a dimension of the table has changed "+
			"and the figure in the doc comment above no longer describes it", pairs, expected)
	}
}

// TestTheComparatorsDoNotContradictEachOther holds the timing comparators to each
// other rather than to expected answers.
//
// Two of the defects this area produced were not wrong answers but impossible
// ones: `@2019-01-01 1 second or less before @2019-01-02` and the same pair with
// `or more` were both true, because a unit finer than the value's precision put
// the bound on the value itself. No table of expected results catches that — the
// two answers are individually plausible. What catches it is asking whether they
// can both hold.
func TestTheComparatorsDoNotContradictEachOther(t *testing.T) {
	lefts := []string{
		"@2019-01-01T00:00:00", "@2019-05-31T23:00:00", "@2018-01-01T00:00:00",
		"@2019-01-01", "@T01:00:00", "@2019-06-01T00:00:00",
	}
	rights := []string{"@2019-06-01T00:00:00", "@2019-01-02", "@T10:00:00"}
	quantities := []string{"1 second", "30 minutes", "3 hours", "1 day", "1 month", "10 years"}
	relationships := []string{"before", "on or before", "after", "on or after"}

	for _, left := range lefts {
		for _, right := range rights {
			for _, q := range quantities {
				for _, rel := range relationships {
					// The inclusive qualifier follows the quantity and the strict one
					// precedes it — two rules in the grammar, not one with a movable
					// word. Writing both the same way made every strict form a syntax
					// error, which is what left three of the four invariants below
					// unreachable.
					inclusive := func(comparator string) string {
						return evalTimingPhrase(t, fmt.Sprintf("%s %s %s %s %s", left, q, comparator, rel, right))
					}
					strict := func(comparator string) string {
						return evalTimingPhrase(t, fmt.Sprintf("%s %s %s %s %s", left, comparator, q, rel, right))
					}
					orLess, orMore := inclusive("or less"), inclusive("or more")
					lessThan, moreThan := strict("less than"), strict("more than")
					where := fmt.Sprintf("%s %s ... %s %s", left, q, rel, right)

					// The strict bounds are the inclusive ones minus the bound
					// itself, so each implies its inclusive twin.
					if lessThan == "true" && orLess != "true" {
						t.Errorf("%s: `less than` holds but `or less` is %s — the strict bound is inside the inclusive one",
							where, orLess)
					}
					if moreThan == "true" && orMore != "true" {
						t.Errorf("%s: `more than` holds but `or more` is %s", where, orMore)
					}
					// And they exclude each other outright.
					if lessThan == "true" && moreThan == "true" {
						t.Errorf("%s: strictly nearer and strictly further at once", where)
					}
					// Both inclusive bounds holding means the distance is the bound
					// exactly, which the offset written without a comparator asks.
					if orLess == "true" && orMore == "true" {
						exact := evalTimingPhrase(t, fmt.Sprintf("%s %s %s %s", left, q, rel, right))
						if exact != "true" {
							t.Errorf("%s: `or less` and `or more` both hold, so the distance is exactly %s — but the exact form is %s",
								where, q, exact)
						}
					}
				}
			}
		}
	}
}
