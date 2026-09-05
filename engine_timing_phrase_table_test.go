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
		return "ERROR: " + err.Error()
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
// 720 pairs. It found `within` unimplemented on its first run — see
// TestWithinStillIgnoresItsQuantity.
func TestEverySpellingOfATimingPhraseAgrees(t *testing.T) {
	lefts := map[string]string{
		"closed bounds": "Interval[@2019-03-01T00:00:00, @2019-09-01T00:00:00]",
		"open bounds":   "Interval(@2019-03-01T00:00:00, @2019-09-01T00:00:00)",
		"dates":         "Interval[@2019-03-01, @2019-09-01]",
	}
	rights := map[string]string{
		"a point inside": "@2019-06-01T00:00:00",
		"a point after":  "@2019-11-01T00:00:00",
		"an interval":    "Interval[@2019-01-01T00:00:00, @2019-12-31T00:00:00]",
		"a date":         "@2019-06-01",
	}
	relationships := []string{"before", "after", "on or before", "on or after"}
	// The strict qualifier comes before the quantity and the inclusive one after
	// it, so each is written the way its own rule spells it.
	offsets := []string{
		"", "1 month or less ", "1 month or more ",
		"less than 1 month ", "more than 1 month ", "6 months or less ",
	}

	pairs := 0
	for leftName, left := range lefts {
		for rightName, right := range rights {
			for _, rel := range relationships {
				for _, off := range offsets {
					// The boundary word names an end, and naming it is the same as
					// taking that end first.
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

					// `occurs` is the default written out, so writing it changes
					// nothing.
					phrase := fmt.Sprintf("%s occurs %s%s %s", left, off, rel, right)
					bare := fmt.Sprintf("%s %s%s %s", left, off, rel, right)
					pairs++
					if a, b := evalTimingPhrase(t, phrase), evalTimingPhrase(t, bare); a != b {
						t.Errorf("[%s, %s] `%s` = %s but `%s` = %s — occurs is the default said aloud",
							leftName, rightName, phrase, a, bare, b)
					}
				}
			}
		}
	}
	if pairs < 700 {
		t.Errorf("only %d pairs compared; the table has stopped covering what it claims to", pairs)
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
					ask := func(comparator string) string {
						return evalTimingPhrase(t, fmt.Sprintf("%s %s %s %s %s", left, q, comparator, rel, right))
					}
					orLess, orMore := ask("or less"), ask("or more")
					lessThan, moreThan := ask("less than"), ask("more than")
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

// TestWithinStillIgnoresItsQuantity records a phrase this table found and this
// change does not fix, asserted rather than described so that fixing it trips the
// test instead of leaving a stale note behind.
//
// `A within 2 months of B` is the same shape of defect the quantity offset had:
// the builder marks the phrase kind and never reads the quantity. Its spellings
// disagree, which is how the table surfaced it —
//
//	Interval[…] starts within 2 months of @2019-07-01     false
//	start of Interval[…] within 2 months of @2019-07-01    null
//
// — and a point one month away answers null rather than true.
//
// It is left alone deliberately: no library in cqframework/ecqm-content-r4 writes
// `within ... of`, and neither does the conformance corpus, so nothing is measured
// to be wrong today. The previous change in this area grew from one defect to the
// whole family and took three review rounds to settle; this one stays a table.
//
// When it is fixed, delete this and let the table above cover the phrase.
func TestWithinStillIgnoresItsQuantity(t *testing.T) {
	const near = "@2019-06-01T00:00:00 within 2 months of @2019-07-01T00:00:00"
	if got := evalTimingPhrase(t, near); got != "null" {
		t.Fatalf("`within` answers %s now — it reads its quantity, so remove this test and "+
			"let TestEverySpellingOfATimingPhraseAgrees cover the phrase", got)
	}

	const asPhrase = "Interval[@2019-06-01T00:00:00, @2019-08-01T00:00:00] starts within 2 months of @2019-07-01T00:00:00"
	const asExtraction = "start of Interval[@2019-06-01T00:00:00, @2019-08-01T00:00:00] within 2 months of @2019-07-01T00:00:00"
	if a, b := evalTimingPhrase(t, asPhrase), evalTimingPhrase(t, asExtraction); a == b {
		t.Fatalf("the two spellings of `within` agree now (%s) — remove this test", a)
	}
}
