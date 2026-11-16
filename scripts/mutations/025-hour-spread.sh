#!/bin/sh
# Mutations of what plan 025 holds (docs/plans/025-hour-spread-by-offset.md): a
# DateTime to the hour that an offset cuts in two is its span of instants; a
# comparison at a precision is asked of every result its pair can come to (the
# window of an unplaced DateTime, the corners of a span), filled between the least
# and the greatest, and answers where they agree; and in, includes, meets and the
# phrases with a quantity are composed from those comparisons.
# Each is applied to a copy of the repository; the command it names has to fail,
# and its output has to say why.
# Usage: sh scripts/mutations/025-hour-spread.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-025.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, re, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-failfast", "-run", "^(" + "|".join(names) + ")$", package]


EV = "eval/evaluator.go"
CS = "eval/compare_set.go"
IP = "eval/interval_precision.go"
PROBES = run("TestAnHourCutByAnOffsetIsItsSpan")
BOUND = run("TestAnOpenBoundOfAnIntervalIsItsSuccessorBetweenIntervals")
QUANTITY = run("TestAQuantityPhraseAppliesItsPredicateToEveryReading")
WINDOW = run("TestTheWindowOfAnUnplacedValueGivesASetOfResults")
M3 = run("TestAnUnplacedValueInAnIntervalIsUnknownWhereTheWindowDecidesNothing")
ORACLE = run("TestTheFractionalOffsetAreaIsItsOracle")
SNAPSHOT = run("TestAnswersMatchTheSnapshots")
PRECISION = run("TestAStatedPrecisionIsTheComparison")
PART = run("TestAPartThatIsOutDecidesWhereTheOtherIsUnknown")
BETWEEN = run("TestAnUnplacedValueInAnIntervalIsUnknownBetweenIntervals")
SPELLINGS = run("TestThePointSpellingsOfMembershipAgree")
PROPERLY = run("TestProperlyIncludesIsStrictOnTheEffectiveBounds")
NULLBOUND = run("TestANullBoundAtAPrecision")
# The engine against the answers it gave to the area: what an oracle row is held to.
AREA = ["go", "test", "-count=1", "-failfast", "-run", "^TestAnswersMatchTheSnapshots$/^fractionaloffset$", "."]
MOVED = "moved in fractionaloffset"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The span.
    ("a spread is read by its first instant",
     (CS, "if !isDateTime || in == nil ||", "if true || !isDateTime || in == nil ||"), PROBES,
     "want"),
    ("a spread is read by its first instant: the area",
     (CS, "if !isDateTime || in == nil ||", "if true || !isDateTime || in == nil ||"), AREA,
     MOVED),
    ("every hour is a spread",
     (CS, "read.Minute() == 0 && read.Second() == 0", "read.Minute() == 99 && read.Second() == 0"), PROBES,
     "want"),
    ("the span is a millisecond too long",
     (CS, "return first, first.Add(time.Hour - time.Millisecond), true", "return first, first.Add(time.Hour), true"), AREA,
     MOVED),
    ("the corners of a span are not filled",
     (CS, "return set.filled(), true", "return set, true"), PROBES,
     "want"),
    # The predicate over the set.
    ("a predicate answers for any member of the set",
     (CS, "\tcase yes && !no:\n\t\treturn fptypes.NewBoolean(true)", "\tcase yes:\n\t\treturn fptypes.NewBoolean(true)"), PROBES,
     "want"),
    ("the predicate is replaced by one result: the window demands the same one at its ends",
     (CS, "\t\t\treturn set.filled()\n\t\t}\n\t}\n\treturn temporalCompareRead",
      "\t\t\tif m := set.seen; set.uncertain || m[0] && m[1] || m[0] && m[2] || m[1] && m[2] {\n\t\t\t\treturn comparisons{uncertain: true}\n\t\t\t}\n\t\t\treturn set\n\t\t}\n\t}\n\treturn temporalCompareRead"),
     WINDOW, "want true"),
    ("the predicate is replaced by one result: the spread demands the same one",
     (CS, "\treturn set.filled(), true\n}", "\tif m := set.seen; set.uncertain || m[0] && m[1] || m[0] && m[2] || m[1] && m[2] {\n\t\treturn comparisons{uncertain: true}, true\n\t}\n\treturn set, true\n}"),
     PROBES, "want"),
    ("the window is not filled",
     (CS, "\t\t\treturn set.filled()\n\t\t}\n\t}\n\treturn temporalCompareRead", "\t\t\treturn set\n\t\t}\n\t}\n\treturn temporalCompareRead"),
     WINDOW, "want null"),
    ("an uncertain part is dropped",
     (CS, "\ts.uncertain = s.uncertain || o.uncertain\n", "\ts.uncertain = s.uncertain && o.uncertain\n"), AREA,
     MOVED),
    # In, includes, meets and the phrases with a quantity.
    ("`in` at a precision does not resolve the window",
     (EV, "\tlow, err := boundsCompare(compare, frame, lowBound, point, precision)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tcompare = temporalBefore\n\tif iv.HighClosed {\n\t\tcompare = temporalSameOrBefore\n\t}\n\thigh, err := boundsCompare(compare, frame, point, highBound, precision)", "\t_ = compare\n\tlow, err := temporalCompareRead(frame, lowBound, point, precision, false).holds(func(c int) bool { return c < 0 || (iv.LowClosed && c == 0) }), error(nil)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tcompare = temporalBefore\n\tif iv.HighClosed {\n\t\tcompare = temporalSameOrBefore\n\t}\n\t_ = compare\n\thigh, err := temporalCompareRead(frame, point, highBound, precision, false).holds(func(c int) bool { return c < 0 || (iv.HighClosed && c == 0) }), error(nil)"),
     M3, "want null"),
    ("`includes` between intervals does not resolve the window",
     (EV, "\tlow, err := boundsCompare(temporalSameOrBefore, frame, effectiveStart(outer), effectiveStart(inner), precision)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\thigh, err := boundsCompare(temporalSameOrBefore, frame, effectiveEnd(inner), effectiveEnd(outer), precision)", "\tlow, err := temporalCompareRead(frame, effectiveStart(outer), effectiveStart(inner), precision, false).holds(func(c int) bool { return c <= 0 }), error(nil)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\thigh, err := temporalCompareRead(frame, effectiveEnd(inner), effectiveEnd(outer), precision, false).holds(func(c int) bool { return c <= 0 }), error(nil)"),
     BETWEEN, "want null"),
    ("`meets` reads a spread as one hour",
     (IP, "if first, last, spread := hourSpread(prev, in); spread {", "if first, last, spread := hourSpread(prev, in); spread && false {"), AREA,
     MOVED),
    ("`meets` reads a spread as a unit",
     (IP, "prevLow, prevHigh, prevOne = componentsOf(first.In(in)), componentsOf(last.In(in)), false", "prevLow, prevHigh, prevOne = componentsOf(first.In(in)), componentsOf(last.In(in)), true"), AREA,
     MOVED),
    ("the early null is back between intervals",
     (EV, "\tincluded := ThreeValuedAnd(low, high)\n", "\tif low == nil || high == nil {\n\t\treturn nil, nil\n\t}\n\tincluded := ThreeValuedAnd(low, high)\n"), PART,
     "want false"),
    ("the early null is back against a point",
     (EV, "\treturn ThreeValuedAnd(low, high), nil\n}\n\n// evalIntervalProperlyContainsPoint", "\tif low == nil || high == nil {\n\t\treturn nil, nil\n\t}\n\treturn ThreeValuedAnd(low, high), nil\n}\n\n// evalIntervalProperlyContainsPoint"), PART,
     "want false"),
    ("a phrase with a quantity asks one result",
     (EV, "\t\treturn temporalCompareSet(e.evaluationFrame(), a, b, precision)\n\t}\n\tcmp, err",
      "\t\tset := temporalCompareSet(e.evaluationFrame(), a, b, precision)\n\t\tif m := set.seen; set.uncertain || m[0] && m[1] || m[0] && m[2] || m[1] && m[2] {\n\t\t\tset.uncertain = true\n\t\t}\n\t\treturn set\n\t}\n\tcmp, err"),
     QUANTITY, "want true"),
    ("between intervals an open bound compares exclusive",
     (EV, "\tincluded := ThreeValuedAnd(low, high)\n", "\tif !outer.LowClosed && inner.LowClosed && outer.Low != nil && inner.Low != nil {\n\t\tlow, _ = temporalBefore(frame, outer.Low, inner.Low, precision)\n\t}\n\tincluded := ThreeValuedAnd(low, high)\n"),
     BOUND, "want true"),
    ("against a point an open bound compares inclusive",
     (EV, "\tcompare := temporalBefore\n\tif iv.LowClosed {", "\tcompare := temporalSameOrBefore\n\tif iv.LowClosed {"), BOUND,
     "want false"),
    # Code review, round 1: the spellings of membership against a point.
    ("includes a point reads it as an interval",
     (EV, "\tif n.Operator.Precision != \"\" && !n.Operator.Properly && n.Operator.Boundary == \"\" {",
      "\tif false && n.Operator.Precision != \"\" && !n.Operator.Properly && n.Operator.Boundary == \"\" {"), SPELLINGS,
     "want false"),
    ("an open start against a point compares inclusive at the hour only",
     (EV, "\tcompare := temporalBefore\n\tif iv.LowClosed {", "\tcompare := temporalBefore\n\tif iv.LowClosed || precision == \"hour\" {"), SPELLINGS,
     "want false"),
    ("an open end against a point compares inclusive at the hour only",
     (EV, "\tcompare = temporalBefore\n\tif iv.HighClosed {", "\tcompare = temporalBefore\n\tif iv.HighClosed || precision == \"hour\" {"), SPELLINGS,
     "want false"),
    # Code review, round 2: properly on the effective bounds.
    ("properly reads a start that is the same as larger",
     (EV, "earlier, err := boundsCompare(temporalBefore, frame, effectiveStart(outer)", "earlier, err := boundsCompare(temporalSameOrBefore, frame, effectiveStart(outer)"), PROPERLY,
     "want false"),
    ("properly compares the written bounds",
     (EV, "earlier, err := boundsCompare(temporalBefore, frame, effectiveStart(outer), effectiveStart(inner), precision)", "earlier, err := boundsCompare(temporalBefore, frame, outer.Low, inner.Low, precision)"), PROPERLY,
     "want false"),
    # Code review, round 3: null bounds and the `and` of properly.
    ("an unknown bound is skipped as true",
     (EV, "\tif a == nil || b == nil {\n\t\treturn nil, nil\n\t}\n\treturn compare(frame, a, b, precision)", "\tif a == nil || b == nil {\n\t\treturn fptypes.NewBoolean(true), nil\n\t}\n\treturn compare(frame, a, b, precision)"), NULLBOUND,
     "want"),
    ("a closed null against a point is unknown",
     (EV, "\tif lowBound == nil && iv.LowClosed {", "\tif false && lowBound == nil && iv.LowClosed {"), NULLBOUND,
     "want true"),
    ("properly returns early on an unknown inclusion",
     (EV, "\tif isFalse(included) {", "\tif !isTrue(included) {"), NULLBOUND,
     "want false"),
    # The expected answers.
    ("an answer of the oracle is corrupted",
     ("testdata/fractionaloffset/oracle.tsv", "D1445\tBoolean false", "D1445\tBoolean true"), ORACLE,
     "D1445"),
    ("an expected answer of a probe is corrupted",
     ("engine_hour_spread_test.go",
      '''{"@2020-01-01T00Z after minute of @2020-01-01T00:10+01:00",
			[4]string{"true", "true", "true", "true"}}''',
      '''{"@2020-01-01T00Z after minute of @2020-01-01T00:10+01:00",
			[4]string{"false", "true", "true", "true"}}'''), PROBES,
     "-10800s"),
    ("temporal D22 is true again",
     ("testdata/snapshots/temporal.answers.tsv", "D22\tnull", "D22\tBoolean true"), run("TestAnswersMatchTheSnapshots"),
     "D22"),
]

# A deliberate verdict of the area names its row in plan 025 (TestTheDivergencesAreConsistent).
plan = open(os.path.join(repo, "docs/plans/025-hour-spread-by-offset.md")).read()
divergences = [l.split("\t") for l in open(os.path.join(repo, "testdata/snapshots/fractionaloffset.divergences.tsv")).read().split("\n")
               if l and not l.startswith("#")]
named_once = [d[0] for d in divergences if d[2] == "deliberate" and "docs/plans/025-hour-spread-by-offset.md" in d[4]
              and len(re.findall(r"(?<![\w-])" + d[0] + r"(?![\w-])", plan)) == 1 and f" {d[0]}," in plan]
if not named_once:
    print("NO DELIBERATE ROW of plan 025 to mutate, named once in the plan as ` Dn,`")
    sys.exit(2)
mutations.append(
    ("a deliberate row that cites a plan that does not name it",
     ("docs/plans/025-hour-spread-by-offset.md", f" {named_once[0]},", " DX,"), run("TestTheDivergencesAreConsistent"),
     "which does not name " + named_once[0]))

CLEAN_ENV = {k: v for k, v in os.environ.items() if not k.startswith("GITHUB_")}


def fresh_copy():
    copy = os.path.join(work, "copy")
    shutil.rmtree(copy, ignore_errors=True)
    for f in files:
        src = os.path.join(repo, f)
        if os.path.isfile(src):
            os.makedirs(os.path.dirname(os.path.join(copy, f)), exist_ok=True)
            shutil.copy2(src, os.path.join(copy, f))
    return copy


# Every command passes on an unmutated copy, or the mutations prove nothing.
baseline = fresh_copy()
for command in {tuple(m[2]) for m in mutations}:
    r = subprocess.run(list(command), cwd=baseline, capture_output=True, text=True, env=CLEAN_ENV)
    if r.returncode != 0:
        print(f"BASELINE FAILS: {' '.join(command)}\n{r.stdout[-2000:]}{r.stderr[-2000:]}")
        sys.exit(2)
print("baseline  passes")

survived = 0
for name, (path, old, new), command, text in mutations:
    copy = fresh_copy()
    target = os.path.join(copy, path)
    source = "\n" + open(target).read()
    if source.count(old) != 1:
        print(f"NOT APPLIED  {name}: {path}: found {source.count(old)} times")
        survived += 1
        continue
    open(target, "w").write(source.replace(old, new)[1:])
    r = subprocess.run(command, cwd=copy, capture_output=True, text=True, env=CLEAN_ENV)
    output = r.stdout + r.stderr
    if "build failed" in output or "[setup failed]" in output:
        print(f"BUILD FAILED {name}: a mutation that does not compile proves nothing\n{output[-1500:]}")
        survived += 1
    elif r.returncode == 0 or text not in output:
        print(f"SURVIVED     {name}: expected a failure saying {text!r}\n{output[-1500:]}")
        survived += 1
    else:
        print(f"caught       {name}")
sys.exit(1 if survived else 0)
PY
