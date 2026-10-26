#!/bin/sh
# Mutations of what plan 008 puts in to move to fhirpath v1.16.6
# (docs/plans/008-fhirpath-1-16.md): the rule that keeps a FHIR value apart
# from a System one, the unit of a sum, the leap second CQL does not have, the
# calendar durations that have no sum, the partial offsets at hour precision,
# and the three areas written by hand that hold them. Each is applied to a copy
# of the repository; the command it names has to fail, and its output has to
# say why.
# Usage: sh scripts/mutations/008-fhirpath.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-008.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


SNAPSHOTS = run("TestAnswersMatchTheSnapshots")
REFERENCES = run("TestAnswersMatchTheReferences")
CONSISTENT = run("TestTheDivergencesAreConsistent")
TYPES = ["go", "test", "-count=1", "-run", "^TestListsAndTuplesKeepModelAndSystemValuesApart$", "./types/"]
EVAL = ["go", "test", "-count=1", "-run", "^TestAModelValueIsNotEqualToASystemValue$", "./eval/"]
CLOCK = "types/clock.go"
SUM = "types/calendar_sum.go"
RULE = "types/model_system.go"
PLAN = "docs/plans/008-fhirpath-1-16.md"


def moved(area):
    return f"answers moved in {area}"


# name, (path, old, new), command, text the failure has to hold
mutations = [
    # A FHIR value against a System one.
    ("a FHIR value equal to a System one",
     (RULE, "\treturn aModel != bModel", "\treturn false && aModel != bModel"), SNAPSHOTS, moved("crossmodel")),
    ("the rule left out of the lists and the tuples",
     ("types/interval.go", "if a == nil || b == nil || ModelAgainstSystem(a, b) {", "if a == nil || b == nil {"),
     TYPES, "is true"),
    ("the rule left out of the element equality",
     ("eval/evaluator.go", "\tif cqltypes.ModelAgainstSystem(a, b) {\n\t\treturn elementsUnequal\n\t}\n", ""),
     EVAL, "FAIL"),
    ("the rule left out of the operations that hold values",
     ("eval/evaluator.go", "\tif cqltypes.ModelAgainstSystem(a, b) {\n\t\treturn false\n\t}\n", ""),
     SNAPSHOTS, moved("crossmodel")),
    # The unit of a sum.
    ("a sum in the unit of its left operand",
     (SUM, "\treturn l.Add(r)\n}",
      "\tq, err := l.Add(r)\n\tif c, ok := q.ConvertTo(l.Unit()); err == nil && ok {\n\t\treturn c, nil\n\t}\n\treturn q, err\n}"),
     SNAPSHOTS, moved("units")),
    # Calendar durations.
    ("a month added to a day",
     (SUM, "\treturn yearsAndMonths[a] && belowTheMonth[b]", "\treturn false && yearsAndMonths[a] && belowTheMonth[b]"),
     SNAPSHOTS, moved("units")),
    ("a week below the month no longer",
     (SUM, '"week": true, "weeks": true, ', ""), SNAPSHOTS, moved("units")),
] + [
    (f"{k} out of the calendar keywords", (SUM, f'"{k}": true', f'"{k}-gone": true'), SNAPSHOTS, moved("units"))
    for k in ["year", "years", "month", "months", "week", "weeks", "day", "days", "hour", "hours", "minute",
              "minutes", "second", "seconds", "millisecond", "milliseconds"]
] + [
    ("Sum adds a month to a day",
     ("eval/aggregate_quantity.go", "cqltypes.AddQuantities(total, q, false)",
      "(func() (fptypes.Quantity, error) { _ = cqltypes.AddQuantities; return total.Add(q) })()"), SNAPSHOTS, moved("units")),
    ("Width subtracts a day from a month",
     ("funcs/interval_set.go", "cqltypes.AddQuantities(hq, lq, true)", "hq.Subtract(lq)"), SNAPSHOTS, moved("units")),
    ("a spread in the unit of its first value",
     ("eval/aggregate_quantity.go", "return fptypes.NewQuantityFromDecimal(root, unit), nil",
      "return fptypes.NewQuantityFromDecimal(root, quantities[0].Unit()+unit[:0]), nil"), SNAPSHOTS, moved("units")),
    # The leap second.
    ("a DateTime with a second of 60",
     (CLOCK, "\tif dt.Second() > 59 {", "\tif dt.Second() > 60 {"), SNAPSHOTS, moved("temporal")),
    ("a Time with a second of 60",
     (CLOCK, "\tif t.Second() > 59 {", "\tif t.Second() > 60 {"), SNAPSHOTS, moved("temporal")),
    ("a DateTime literal read by fhirpath",
     ("eval/evaluator.go", "dt, err := cqltypes.NewDateTime(dtVal)", "dt, err := fptypes.NewDateTime(dtVal)"),
     SNAPSHOTS, moved("temporal")),
    ("a Time literal read by fhirpath",
     ("eval/evaluator.go", "t, err := cqltypes.NewTime(timeVal)", "t, err := fptypes.NewTime(timeVal)"),
     SNAPSHOTS, moved("temporal")),
    ("ToDateTime read by fhirpath",
     ("funcs/type_ops.go", "dt, err := cqltypes.NewDateTime(val.Value())", "dt, err := fptypes.NewDateTime(val.Value())"),
     SNAPSHOTS, moved("temporal")),
    ("ToTime read by fhirpath",
     ("funcs/type_ops.go", "t, err := cqltypes.NewTime(sv)", "t, err := fptypes.NewTime(sv)"), SNAPSHOTS, moved("temporal")),
    ("the DateTime constructor read by fhirpath",
     ("funcs/datetime_advanced.go", "return cqltypes.NewDateTime(s)", "return fptypes.NewDateTime(s)"),
     SNAPSHOTS, moved("temporal")),
    ("the Time constructor read by fhirpath",
     ("funcs/datetime_advanced.go", "return cqltypes.NewTime(s)", "return fptypes.NewTime(s)"), SNAPSHOTS, moved("temporal")),
    ("a leap second read from FHIR data",
     ("eval/planned_type.go", "if isDateTime && dt.Second() > 59 {", "if isDateTime && dt.Second() > 60 {"),
     run("TestALeapSecondReadFromFHIRDataIsNoCQLValue"), "want <nil>"),
    ("ConvertsToDateTime read by fhirpath",
     ("eval/evaluator.go", "_, parseErr := cqltypes.NewDateTime(v.Value())", "_, parseErr := fptypes.NewDateTime(v.Value())"),
     SNAPSHOTS, moved("temporal")),
    ("convert to DateTime read by fhirpath",
     ("eval/evaluator.go", "dt, err := cqltypes.NewDateTime(s.Value())", "dt, err := fptypes.NewDateTime(s.Value())"),
     SNAPSHOTS, moved("temporal")),
    ("convert to Time read by fhirpath",
     ("eval/evaluator.go", "\t\t\treturn cqltypes.NewTime(str)", "\t\t\treturn fptypes.NewTime(str)"),
     SNAPSHOTS, moved("temporal")),
    # Partial offsets.
    ("partial offsets compared as fhirpath does",
     ("types/temporal_compare.go", "\tif cmp, applies, err := compareAcrossPartialOffsets(a, b); applies {\n\t\treturn cmp, err\n\t}\n", ""),
     SNAPSHOTS, moved("temporal")),
    ("partial offsets ordered where the hours overlap",
     ("types/temporal_compare.go", "\treturn 0, true, fptypes.ErrPrecisionMismatch\n}", "\treturn -1, true, nil\n}"),
     SNAPSHOTS, moved("temporal")),
    ("whole-hour offsets read as partial",
     ("types/temporal_compare.go", "(leftOffset-rightOffset)%60 == 0", "(leftOffset-rightOffset)%60 == 1"),
     SNAPSHOTS, "answers moved in"),
    ("partial offsets only between two hours",
     ("types/temporal_compare.go", "left.Precision() != fptypes.DTHourPrecision && right.Precision() != fptypes.DTHourPrecision",
      "left.Precision() != fptypes.DTHourPrecision || right.Precision() != fptypes.DTHourPrecision"),
     SNAPSHOTS, moved("temporal")),
    ("the span of an hour as long as a minute",
     ("types/temporal_compare.go", "return start, start.Add(time.Hour)", "return start, start.Add(time.Minute)"),
     SNAPSHOTS, moved("temporal")),
    ("the span of a second as long as a minute",
     ("types/temporal_compare.go", "return start, start.Add(time.Second)", "return start, start.Add(time.Minute)"),
     SNAPSHOTS, moved("temporal")),
    ("the later span ordered a minute late",
     ("types/temporal_compare.go", "case !rightEnd.After(leftStart):", "case !rightEnd.After(leftStart.Add(-time.Minute)):"),
     SNAPSHOTS, moved("temporal")),
    # The three areas.
    ("a deliberate divergence dropped from its file",
     ("testdata/snapshots/units.divergences.tsv", "\nD0\tjava\tdeliberate\t", "\nD0-gone\tjava\tdeliberate\t"),
     REFERENCES, "are not the ones recorded"),
    ("a deliberate divergence the plan does not name",
     (PLAN, "D69, D70, D71 y D72.", "D69, D71 y D72."), CONSISTENT, "does not name D70"),
    ("an own defect expected as this engine answers",
     ("testdata/snapshots/units.divergences.tsv", "\nD25\tjava\town defect\t100.00000001 'cm'\t", "\nD25\tjava\town defect\t100 'cm'\t"),
     CONSISTENT, "an own defect expects an answer other than this engine's"),
    ("crossmodel run without the FHIR data",
     ("engine_snapshot_test.go", ' || area == "crossmodel"\n', "\n"), SNAPSHOTS, moved("crossmodel")),
    ("an area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"units", 73, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "units.cql is not in snapshotAreas"),
    ("an area with another number of definitions",
     ("engine_snapshot_test.go", '{"temporal", 49, true},', '{"temporal", 48, true},'), SNAPSHOTS, "the manifest says 48"),
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
