package cql

import (
	"regexp"
	"testing"
	"time"

	fptypes "github.com/gofhir/fhirpath/types"
)

// Plan 024 (docs/plans/024-same-precision-offset.md): a comparison at a
// precision of the hour or finer reads two DateTimes in what they wrote when
// their offsets are the same, and both in the offset of the evaluation request
// when they differ. A DateTime that writes no offset has the evaluation's.
//
// CQL 1.5.3, Appendix B, Same As (https://cql.hl7.org/N1A/09-b-cqlreference.html):
// "When comparing DateTime values with different timezone offsets,
// implementations should normalize to the timezone offset of the evaluation
// request timestamp, but only when the comparison precision is hours, minutes,
// seconds, or milliseconds." Same Or Before, Same Or After, Before and After
// say the same.

const (
	plus0530  = 5*3600 + 30*60
	plus0545  = 5*3600 + 45*60
	minus0930 = -(9*3600 + 30*60)
	minus0300 = -3 * 3600
	utcOffset = 0
)

// answerText is an answer as the table writes it.
func answerText(v fptypes.Value) string {
	if v == nil {
		return "null"
	}
	return v.String()
}

// TestAHourIsCutInTheEvaluationOffset: the probes of the plan, with what the
// Java engine 5.4.0 answers (the oracle where a value writes no offset:
// cql-execution truncates a +05:30 evaluation offset to +05:00, plan 024, A3)
// and cql-execution 3.3.2 where it agrees.
func TestAHourIsCutInTheEvaluationOffset(t *testing.T) {
	type at struct {
		offset int
		want   string
	}
	for _, tt := range []struct {
		name, expr string
		wants      []at
	}{
		// D6: both at the evaluation's offset, whatever it is, so the same hour.
		// Java and cql-execution: true at +05:30, +05:45 and -03:00.
		{"D6", "@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00",
			[]at{{plus0530, "true"}, {plus0545, "true"}, {minus0930, "true"}, {minus0300, "true"}, {utcOffset, "true"}}},
		// D20: not before at the hour. Java and cql-execution: false.
		{"D20", "@2020-01-01T00:10:00 before hour of @2020-01-01T00:40:00",
			[]at{{plus0530, "false"}, {plus0545, "false"}, {minus0930, "false"}, {minus0300, "false"}, {utcOffset, "false"}}},
		// D21: 00:40+05:30 is 00:40 at +05:30, 00:55 at +05:45 and 19:10 on the
		// day before at -03:00. Java: true, true, false; at -09:30 it is 09:40 on
		// the day before, by the same reading.
		{"D21", "@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00+05:30",
			[]at{{plus0530, "true"}, {plus0545, "true"}, {minus0930, "false"}, {minus0300, "false"}, {utcOffset, "false"}}},
		// D8 of the attack: the hour of the end and the hour of the start are the
		// same, so the intervals do not meet. Java and cql-execution: false.
		{"D8", "Interval[@2020-01-01T00:10:00, @2020-01-01T00:20:00] meets hour of Interval[@2020-01-01T00:40:00, @2020-01-01T01:00:00]",
			[]at{{plus0530, "false"}, {plus0545, "false"}, {minus0930, "false"}, {minus0300, "false"}, {utcOffset, "false"}}},
		// D5: written at Z next to one with no offset, which is the evaluation's.
		// Java and cql-execution: false at +05:30, +05:45 and -03:00. At Z the
		// two are the same offset, and the same hour.
		{"D5", "@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00",
			[]at{{plus0530, "false"}, {plus0545, "false"}, {minus0930, "false"}, {minus0300, "false"}, {utcOffset, "true"}}},
		// D19 of the attack, Java: true at +05:30, where 00:10 with no offset and
		// 00:10+05:30 are the same minute.
		{"D19", "@2020-01-01T00:10:00 same minute as @2020-01-01T00:10:00+05:30",
			[]at{{plus0530, "true"}}},
	} {
		for _, w := range tt.wants {
			if got := answerText(evalValueAt(t, w.offset, tt.expr)); got != w.want {
				t.Errorf("%s at %+ds: %s = %s; want %s", tt.name, w.offset, tt.expr, got, w.want)
			}
		}
	}
}

// TestTheSameWrittenOffsetIsReadAsWritten: with one offset on both sides there
// is nothing to normalize, whatever the offset of the evaluation (plan 024,
// round 1, A1: the 1.5.3 normalizes "when comparing DateTime values with
// different timezone offsets"). Java and cql-execution answer true for the
// +01:00 pair at +05:30 (D3 of the attack), and for the Z pair (D14).
func TestTheSameWrittenOffsetIsReadAsWritten(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"@2020-01-01T00:10:00+01:00 same hour as @2020-01-01T00:40:00+01:00", "true"},
		{"@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00Z", "true"},
		{"@2020-01-01T18:20:00Z same hour as @2020-01-01T18:40:00Z", "true"},
		// The written hours are 0 and 1. At +05:30 both are 05:xx once moved
		// there (23:50Z and 00:10Z), so normalizing would say true.
		{"@2020-01-01T00:50:00+01:00 same hour as @2020-01-01T01:10:00+01:00", "false"},
		{"@2020-01-01T00:50:00Z same hour as @2020-01-01T01:10:00Z", "false"},
		{"@2020-01-01T00:50:00+01:00 before hour of @2020-01-01T01:10:00+01:00", "true"},
	} {
		for _, offset := range []int{plus0530, plus0545, minus0930, minus0300, utcOffset} {
			if got := answerText(evalValueAt(t, offset, tt.expr)); got != tt.want {
				t.Errorf("at %+ds: %s = %s; want %s", offset, tt.expr, got, tt.want)
			}
		}
	}
}

// TestDifferentOffsetsAreReadInTheEvaluationOffset: the literal reading of
// Appendix B, which the references do not follow: both read the pair in the
// offset of the left operand, and answer true for the first row at +05:30
// (docs/plans/024-same-precision-offset.md, decision of 14-11-2026).
func TestDifferentOffsetsAreReadInTheEvaluationOffset(t *testing.T) {
	type at struct {
		offset int
		want   string
	}
	for _, tt := range []struct {
		expr  string
		wants []at
	}{
		// 00:10Z and 00:40Z, which are 05:40 and 06:10 at +05:30, 05:55 and 06:25
		// at +05:45, and 14:40 and 15:10 on the day before at -09:30.
		{"@2020-01-01T00:10:00Z same hour as @2020-01-01T01:40:00+01:00",
			[]at{{plus0530, "false"}, {plus0545, "false"}, {minus0930, "false"}, {minus0300, "true"}, {utcOffset, "true"}}},
		{"@2020-01-01T00:10:00Z before hour of @2020-01-01T01:40:00+01:00",
			[]at{{plus0530, "true"}, {plus0545, "true"}, {minus0930, "true"}, {minus0300, "false"}, {utcOffset, "false"}}},
		{"@2020-01-01T01:40:00+01:00 after hour of @2020-01-01T00:10:00Z",
			[]at{{plus0530, "true"}, {plus0545, "true"}, {minus0930, "true"}, {minus0300, "false"}, {utcOffset, "false"}}},
		{"@2020-01-01T00:10:00Z same hour or after @2020-01-01T01:40:00+01:00",
			[]at{{plus0530, "false"}, {minus0300, "true"}}},
		{"@2020-01-01T00:10:00Z same hour or before @2020-01-01T01:40:00+01:00",
			[]at{{plus0530, "true"}, {minus0300, "true"}}},
	} {
		for _, w := range tt.wants {
			if got := answerText(evalValueAt(t, w.offset, tt.expr)); got != w.want {
				t.Errorf("at %+ds: %s = %s; want %s", w.offset, tt.expr, got, w.want)
			}
		}
	}
}

// TestEveryPhraseWithAPrecisionReadsTheHourInTheEvaluationOffset puts each
// operator of the scope on two pairs of times. 00:20 and 00:50 are one local
// hour and, at the fractional offsets, two UTC hours; 00:50 and 01:10 are two
// local hours and one UTC hour. A time with no offset is read at the
// evaluation's, so only the local hour counts. At -03:00 and Z the two readings
// are the same hour, and the answers are the same.
//
// Each expression is asked three ways: with no offset on either side, where the
// digits are read as they are written; and with the instants of one side written
// at Z, on the right and then on the left, where the offsets differ and both are
// read in the offset of the evaluation, so the answer is the same. Only that
// reading puts the frame in play: a caller that passes another one is seen
// here, for each phrase. {l:hh:mm:ss} is a time of the left operand, {r:…} one
// of the right.
func TestEveryPhraseWithAPrecisionReadsTheHourInTheEvaluationOffset(t *testing.T) {
	for _, tt := range []struct{ tpl, want string }{
		{"{l:00:20:00} same hour as {r:00:50:00}", "true"},
		{"{l:00:50:00} same hour as {r:01:10:00}", "false"},
		{"{l:01:10:00} same hour or before {r:00:50:00}", "false"},
		{"{l:00:20:00} same hour or before {r:00:50:00}", "true"},
		{"{l:00:20:00} same hour or after {r:00:50:00}", "true"},
		{"{l:00:50:00} same hour or after {r:01:10:00}", "false"},
		{"{l:00:20:00} before hour of {r:00:50:00}", "false"},
		{"{l:00:50:00} before hour of {r:01:10:00}", "true"},
		{"{l:00:50:00} after hour of {r:00:20:00}", "false"},
		{"{l:01:10:00} after hour of {r:00:50:00}", "true"},
		{"{l:00:20:00} same minute as {r:00:20:30}", "true"},
		{"{l:00:20:00} in hour of Interval[{r:00:50:00}, {r:02:00:00}]", "true"},
		{"{l:00:50:00} in hour of Interval[{r:01:10:00}, {r:02:00:00}]", "false"},
		{"Interval[{l:00:20:00}, {l:00:50:00}] during hour of Interval[{r:00:30:00}, {r:00:30:00}]", "true"},
		{"Interval[{l:00:20:00}, {l:00:50:00}] included in hour of Interval[{r:00:30:00}, {r:00:30:00}]", "true"},
		{"Interval[{l:00:00:00}, {l:00:59:00}] includes hour of Interval[{r:00:20:00}, {r:00:50:00}]", "true"},
		{"Interval[{l:00:00:00}, {l:00:30:00}] includes hour of {r:00:50:00}", "true"},
		{"Interval[{l:00:50:00}, {l:00:50:00}] overlaps hour of Interval[{r:01:10:00}, {r:01:10:00}]", "false"},
		{"Interval[{l:00:50:00}, {l:00:50:00}] starts hour of Interval[{r:01:10:00}, {r:01:10:00}]", "false"},
		{"Interval[{l:00:50:00}, {l:00:50:00}] ends hour of Interval[{r:01:10:00}, {r:01:10:00}]", "false"},
		{"Interval[{l:00:50:00}, {l:00:50:00}] before hour of Interval[{r:01:10:00}, {r:01:10:00}]", "true"},
		{"Interval[{l:01:10:00}, {l:01:10:00}] after hour of Interval[{r:00:50:00}, {r:00:50:00}]", "true"},
		{"Interval[{l:00:10:00}, {l:00:50:00}] meets hour of Interval[{r:01:10:00}, {r:01:20:00}]", "true"},
		{"Interval[{l:01:10:00}, {l:01:20:00}] meets hour of Interval[{r:00:10:00}, {r:00:50:00}]", "true"},
		{"Interval[{l:00:20:00}, {l:00:50:00}] meets hour of Interval[{r:00:55:00}, {r:01:20:00}]", "false"},
		{"Interval[{l:00:10:00}, {l:00:20:00}] meets hour of Interval[{r:00:40:00}, {r:01:00:00}]", "false"},
		{"Interval[{l:00:10:00}, {l:00:50:00}] same hour or before Interval[{r:00:55:00}, {r:01:20:00}]", "true"},
		{"Interval[{l:00:10:00}, {l:00:50:00}] before hour of Interval[{r:01:10:00}, {r:01:20:00}]", "true"},
		{"Interval[{l:00:10:00}, {l:00:50:00}] before hour of Interval[{r:00:55:00}, {r:01:20:00}]", "false"},
		// Interval `same as` at a precision, the properly forms, and the phrase
		// with a quantity offset: each has its own door to the frame.
		{"Interval[{l:00:20:00}, {l:00:50:00}] same hour as Interval[{r:00:55:00}, {r:00:55:00}]", "true"},
		{"Interval[{l:00:50:00}, {l:00:50:00}] same hour as Interval[{r:01:10:00}, {r:01:10:00}]", "false"},
		{"Interval[{l:00:20:00}, {l:02:50:00}] properly includes hour of {r:00:50:00}", "false"},
		{"Interval[{l:00:20:00}, {l:02:50:00}] properly includes hour of {r:01:55:00}", "true"},
		{"Interval[{l:00:10:00}, {l:02:00:00}] starts during hour of Interval[{r:00:50:00}, {r:00:55:00}]", "true"},
		{"Interval[{l:00:00:00}, {l:00:10:00}] ends during hour of Interval[{r:00:50:00}, {r:00:55:00}]", "true"},
		{"{l:00:50:00} properly included in hour of Interval[{r:00:20:00}, {r:02:50:00}]", "false"},
		{"{l:00:20:00} 1 hour or less before hour of {r:00:55:00}", "false"},
		{"{l:00:20:00} 1 hour or less on or before hour of {r:00:55:00}", "true"},
	} {
		for _, offset := range []int{plus0530, plus0545, minus0930, minus0300, utcOffset} {
			for _, way := range []struct {
				name              string
				leftAtZ, rightAtZ bool
			}{
				{"no offsets", false, false}, {"the right at Z", false, true}, {"the left at Z", true, false},
			} {
				expr := phraseTimes.ReplaceAllStringFunc(tt.tpl, func(token string) string {
					m := phraseTimes.FindStringSubmatch(token)
					return timeAt(offset, m[2], m[1] == "l" && way.leftAtZ || m[1] == "r" && way.rightAtZ)
				})
				if got := answerText(evalValueAt(t, offset, expr)); got != tt.want {
					t.Errorf("at %+ds, %s: %s = %s; want %s", offset, way.name, expr, got, tt.want)
				}
			}
		}
	}
}

var phraseTimes = regexp.MustCompile(`\{([lr]):(\d\d:\d\d:\d\d)\}`)

// timeAt is a DateTime of the 1st of January 2020 at a time of the evaluation's
// own offset, written with no offset, or as the same instant at Z.
func timeAt(offset int, clock string, atZ bool) string {
	if !atZ {
		return "@2020-01-01T" + clock
	}
	local, err := time.ParseInLocation("2006-01-02T15:04:05", "2020-01-01T"+clock, time.FixedZone("", offset))
	if err != nil {
		panic(err)
	}
	return "@" + local.UTC().Format("2006-01-02T15:04:05") + "Z"
}

// TestACoarserPrecisionAndADateKeepTheDigitsWritten: nothing changes below the
// hour, nor with a Date or a Time. The pair is on the 1st in the offsets it
// was written at, and on two days at UTC, in neither of the readings.
func TestACoarserPrecisionAndADateKeepTheDigitsWritten(t *testing.T) {
	for _, tt := range []struct{ expr, want string }{
		{"@2020-01-01T23:00:00Z same day as @2020-01-01T21:00:00-03:00", "true"},
		{"@2020-01-01T23:30:00+05:30 same day as @2020-01-01T00:30:00Z", "true"},
		{"@2020-01-01T23:30:00+05:30 same month as @2020-01-31T00:30:00Z", "true"},
		{"@2020-01-01 same day as @2020-01-01T10:00:00", "true"},
		{"@T10:20:00 same hour as @T10:50:00", "true"},
	} {
		for _, offset := range []int{plus0530, plus0545, minus0930, minus0300, utcOffset} {
			if got := answerText(evalValueAt(t, offset, tt.expr)); got != tt.want {
				t.Errorf("at %+ds: %s = %s; want %s", offset, tt.expr, got, tt.want)
			}
		}
	}
}

// TestAnUnplacedValueIsUncertainWhereTheWindowDisagrees: a DateTime parameter
// that writes no offset and has none assumed could be at any offset from -12:00
// to +14:00 (eval/timing_offset.go). Across that window the comparison reads
// both values in the evaluation offset, except at the one offset that equals
// the other value's, where the digits written are compared (plan 024). The
// two readings can disagree, so agreement at the two ends of the window is not
// enough (code review of plan 024, round 1, finding 2).
//
// At +05:30, 00:50 placed at -11:59 is 18:19 on the 1st at +05:30, and
// 01:10-12:00 is 18:40: the same hour, so not before. Placed at -12:00 it has
// the other value's offset, and 00 is before 01. Unknown.
func TestAnUnplacedValueIsUncertainWhereTheWindowDisagrees(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", plus0530))
	for _, tt := range []struct{ p, expr, want string }{
		// The two ends disagree: at +14:00 P is the 31st, at -12:00 the same hour.
		{"2020-01-01T00:50:00", "P before hour of @2020-01-01T01:10:00-12:00", "null"},
		// The two ends agree, before, and the offset between them that equals the
		// other value's does not: 01:05-12:00 is 18:35 at +05:30 and 01:40-12:00
		// is 19:10, but as written both are 01. Not before at that one offset.
		{"2020-01-01T01:05:00", "P before hour of @2020-01-01T01:40:00-12:00", "null"},
		// Twenty-six hours apart: no offset of the window reorders them.
		{"2020-01-01T00:50:00", "P before hour of @2020-01-03T01:10:00-12:00", "true"},
	} {
		unplaced, err := fptypes.NewDateTime(tt.p)
		if err != nil {
			t.Fatal(err)
		}
		source := "library P\nparameter P DateTime\ndefine X: " + tt.expr
		v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(
			t.Context(), source, "X", nil, map[string]fptypes.Value{"P": unplaced})
		if err != nil {
			t.Fatalf("%s: %v", tt.expr, err)
		}
		if got := answerText(v); got != tt.want {
			t.Errorf("P = %s: %s = %s; want %s", tt.p, tt.expr, got, tt.want)
		}
	}
}

// TestAnUnplacedValueInAnIntervalIsReadAsBefore: the readers of components
// behind `in`, `includes` and `during` do not resolve a DateTime that writes no
// offset over the window of offsets, as the comparisons do, and read it as they
// did before plan 024. The answer is a known defect: 00:50 at an offset from
// -12:00 to +14:00 may or may not fall in the hour of 00Z, so it is unknown,
// where main and this engine answer true (plan 024, code review round 3, M3;
// backlog). The test holds that plan 024 does not move it.
func TestAnUnplacedValueInAnIntervalIsReadAsBefore(t *testing.T) {
	unplacedValue, err := fptypes.NewDateTime("2020-01-01T00:50")
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{minus0300, plus0530} {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("", offset))
		v, err := NewEngine(WithEvaluationTimestamp(at)).EvaluateExpression(t.Context(),
			"library P\nparameter P DateTime\ndefine X: P in hour of Interval[@2020-01-01T00:00Z, @2020-01-01T01:00Z]",
			"X", nil, map[string]fptypes.Value{"P": unplacedValue})
		if err != nil {
			t.Fatal(err)
		}
		if got := answerText(v); got != "true" {
			t.Errorf("at %+ds: P in hour of Interval[00Z, 01Z] = %s; want true, as before plan 024", offset, got)
		}
	}
}
