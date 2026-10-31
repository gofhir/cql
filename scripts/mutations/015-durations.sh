#!/bin/sh
# Mutations of what plan 015 holds (docs/plans/015-years-between-leap.md): the
# count of whole calendar periods in years between and months between, with the
# end-of-month sum, the time of the day to the millisecond, the precision both
# operands share, and the sign of the reverse direction; the calendar edges; the
# ages that go through that count; the CalculateAgeIn...At operators; the
# uncertainty read with both bounds; and the area that holds the cases. Each is
# applied to a copy of the repository; the command it names has to fail, and its
# output has to say why.
# Usage: sh scripts/mutations/015-durations.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-015.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


COUNT = run("TestYearsAndMonthsBetweenCountCalendarPeriods", "TestADurationIsTheSameFromEveryEntryPoint",
            "TestTheCountsHoldTheirDefinitionOverTheGrid", "TestDurationBetweenAtLowPrecisionIsAnUncertainty")
AGES = run("TestTheAgeIsADurationInTheUnit", "TestAPatientBornOnTheLeapDayIsAYearOlderOnThe28th")
ANSWERS = run("TestAnUncertaintyIsReadWithBothBounds", package="./internal/referenceharness/answers/")
CONSISTENT = run("TestTheDivergencesAreConsistent")
TEMPORAL = "funcs/temporal.go"
CLINICAL = "funcs/clinical.go"
EVAL = "eval/evaluator.go"
PLAN = "docs/plans/015-years-between-leap.md"
ANSWERS_GO = "internal/referenceharness/answers/answers.go"
AGE_YEARS = "\treturn YearsBetween(birthDate, asOf)\n"
AGE_MONTHS = "\treturn MonthsBetween(birthDate, asOf)\n"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The count.
    ("the day of the month kept, with no end-of-month adjustment",
     (TEMPORAL, "moved.Year(), moved.Month(), moved.Day()\n", "moved.Year(), moved.Month(), low.field[2]\n"), COUNT,
     "years between @2012-02-29 and @2014-02-28 = 1, want 2"),
    ("the time of the day ignored",
     (TEMPORAL, "return compareParts(candidate, high, prec) <= 0\n", "return compareParts(candidate, high, min(prec, 2)) <= 0\n"), COUNT,
     "years between @2012-02-29T12:34:56 and @2014-02-28T12:34:55 = 2, want 1"),
    ("the milliseconds ignored",
     (TEMPORAL, "return compareParts(candidate, high, prec) <= 0\n", "return compareParts(candidate, high, min(prec, 5)) <= 0\n"), COUNT,
     "years between @2012-02-29T12:34:56.001 and @2014-02-28T12:34:56.000 = 2, want 1"),
    ("one period too many in the reverse direction",
     (TEMPORAL, "return -calendarPeriodsBetween(high, low, unit)\n", "return -calendarPeriodsBetween(high, low, unit) - 1\n"), COUNT,
     "years between @2014-02-28 and @2012-02-29 = -3, want -2"),
    ("the finer precision of the two, not the common one",
     (TEMPORAL, "prec := min(low.prec, high.prec)\n", "prec := max(low.prec, high.prec)\n"), COUNT,
     "years between @2012-02-29T23:59:59.999 and @2014-02-28 = 1, want 2"),
    ("the estimate never backed off",
     (TEMPORAL, "if periods > 0 && !periodsFit(", "if false && !periodsFit("), COUNT,
     "years between @2012-02-29T12:34:56 and @2014-02-28T12:34:55 = 2, want 1"),
    ("months counted as years",
     (TEMPORAL, 'if unit == "month" {\n', 'if false {\n'), COUNT,
     "months between @2014-01-01 and @2014-03-15 = 0, want 2"),
    ("Go's zero time as no value",
     (TEMPORAL, "\tif !lok || !hok {\n", "\tif !lok || !hok || l.field == [7]int{1, 1, 1} {\n"), COUNT,
     "years between @0001-01-01 and @0002-01-01 = null, want 1"),
    ("the lower bound of an uncertainty from the other extreme",
     ("funcs/datetime_advanced.go", "minLow := lowLatest\n", "minLow := lowEarliest\n\t_ = lowLatest\n"), COUNT,
     "years between @2012 and @2014 = 2, want Interval[1, 2]"),
    # The ages.
    ("the age in years by a count of its own",
     (CLINICAL, AGE_YEARS, "\tb, _ := partsOf(birthDate)\n\tr, _ := partsOf(asOf)\n\tyears := r.field[0] - b.field[0]\n"
      "\tif r.field[1] < b.field[1] || (r.field[1] == b.field[1] && r.field[2] < b.field[2]) {\n\t\tyears--\n\t}\n"
      "\treturn fptypes.NewInteger(int64(years)), nil\n"), AGES,
     "CalculateAgeInYearsAt(@2012-02-29, @2014-02-28) = 1, want 2"),
    ("the age in months by a count of its own",
     (CLINICAL, AGE_MONTHS, "\tb, _ := partsOf(birthDate)\n\tr, _ := partsOf(asOf)\n\tmonths := (r.field[0]-b.field[0])*12 + r.field[1] - b.field[1]\n"
      "\tif r.field[2] < b.field[2] {\n\t\tmonths--\n\t}\n\treturn fptypes.NewInteger(int64(months)), nil\n"), AGES,
     "CalculateAgeInMonthsAt(@2014-01-31, @2014-02-28) = 0, want 1"),
    ("CalculateAgeInYearsAt not dispatched",
     (EVAL, 'case "calculateageinyears", "calculateageinyearsat":', 'case "calculateageinyears":'), AGES,
     "CalculateAgeInYearsAt(@2012-02-29, @2014-02-28) = error"),
    ("CalculateAgeInMonthsAt not dispatched",
     (EVAL, 'case "calculateageinmonths", "calculateageinmonthsat":', 'case "calculateageinmonths":'), AGES,
     "CalculateAgeInMonthsAt(@2014-01-31, @2014-02-28) = error"),
    ("CalculateAgeInWeeksAt not dispatched",
     (EVAL, 'case "calculateageinweeks", "calculateageinweeksat":', 'case "calculateageinweeks":'), AGES,
     "CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07) = error"),
    ("CalculateAgeInDaysAt not dispatched",
     (EVAL, 'case "calculateageindays", "calculateageindaysat":', 'case "calculateageindays":'), AGES,
     "CalculateAgeInDaysAt(@2012-02-29, @2013-02-28) = error"),
    ("CalculateAgeInYearsAt not typed",
     ("sema/systemfuncs.go", '\t"calculateageinyearsat":  fixed(Integer),\n', ""),
     run("TestTheAgeAtOperatorsAreTypedAsIntegers", package="./sema/"), "Years = ?, want System.Integer"),
    # The harness.
    ("an uncertainty of the reference left unread",
     (ANSWERS_GO, '\tcase "Uncertainty":\n', '\tcase "Uncertainty ":\n'), ANSWERS,
     "an uncertainty and the same one were reported different"),
    ("an uncertainty of the reference with one bound",
     (ANSWERS_GO, "uncertainty(Reference(v.Low), Reference(v.High))", "uncertainty(Reference(v.Low), Reference(v.Low))"), ANSWERS,
     "an uncertainty and the same one were reported different"),
    ("an uncertainty of the engine with one bound",
     (ANSWERS_GO, "uncertainty(Value(n.Low), Value(n.High))", "uncertainty(Value(n.Low), Value(n.Low))"), ANSWERS,
     "an uncertainty and the same one were reported different"),
    # The area and its classification.
    ("a deliberate divergence the plan does not name",
     (PLAN, "D263, D267", "D267"), CONSISTENT, "does not name D263"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"durations", 298, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "durations.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", "conversions durations", "conversions"),
     run("TestEveryAreaIsRunByTheReferences"), "scripts/references.sh does not run it"),
    ("a test of G3b back in the suite's area",
     ("testdata/snapshots/suitegaps.cql", "\ndefine D23:",
      "\ndefine D21: years between @2012-02-29 and @2014-02-28 // YearsBetweenLeapYearDatesEquals2\ndefine D23:"),
     run("TestEveryKnownFailureIsOfTheSuite", package="./conformance/"),
     "holds YearsBetweenLeapYearDatesEquals2, which is not in knownFailures"),
]

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
    source = open(target).read()
    if source.count(old) != 1:
        print(f"NOT APPLIED  {name}: {path}: found {source.count(old)} times")
        survived += 1
        continue
    open(target, "w").write(source.replace(old, new))
    r = subprocess.run(command, cwd=copy, capture_output=True, text=True, env=CLEAN_ENV)
    output = r.stdout + r.stderr
    if "build failed" in output or "[setup failed]" in output:
        print(f"BUILD FAILED {name}: a mutation that does not compile proves nothing")
        survived += 1
    elif r.returncode == 0 or text not in output:
        print(f"SURVIVED     {name}: expected a failure saying {text!r}\n{output[-1500:]}")
        survived += 1
    else:
        print(f"caught       {name}")
sys.exit(1 if survived else 0)
PY
