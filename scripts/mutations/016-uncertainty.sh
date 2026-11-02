#!/bin/sh
# Mutations of what plan 016 holds (docs/plans/016-duration-uncertainty.md): an
# uncertainty asked about each value it could be in between, in, during, included
# in, includes, contains, before, after, same or before, same or after and in a
# list, with the closure
# of each bound and the comparison of two uncertainties by their extremes; the
# range of an incomplete DateTime built in the frame of the value; seconds and
# milliseconds as one precision; and the area that holds the cases. Each is
# applied to a copy of the repository; the command it names has to fail, and its
# output has to say why.
# Usage: sh scripts/mutations/016-uncertainty.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-016.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


HAND = run("TestTheOperatorsOfAnUncertaintyByHand")
GRID = run("TestAnUncertaintyIsAnsweredByItsEndpoints")
OFFSETS = run("TestTheNormativeExamplesDoNotDependOnTheOffset")
CONSISTENT = run("TestTheDivergencesAreConsistent")
EVAL = "eval/uncertain_membership.go"
EVALUATOR = "eval/evaluator.go"
ADVANCED = "funcs/datetime_advanced.go"
PLAN = "docs/plans/016-duration-uncertainty.md"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The membership family.
    ("between without the branch of the uncertainty",
     (EVALUATOR, "if isUncertainty(operand) || isUncertainty(low) || isUncertainty(high) {", "if false {"), HAND,
     "(years between @2012 and @2014) between 0 and 5 = error"),
    ("in and contains without the branch of the uncertainty",
     (EVALUATOR, "if res, handled, err := uncertainContained(x, container, false); handled {", "if res, handled, err := uncertainContained(x, container, false); handled && false {"), HAND,
     "(years between @2012 and @2014) in Interval[0, 5] = error"),
    ("the timing phrases without the branch of the uncertainty",
     (EVALUATOR, "if res, handled, uerr := uncertainTiming(n.Operator, left, right); handled {", "if res, handled, uerr := uncertainTiming(n.Operator, left, right); handled && false {"), HAND,
     "(years between @2012 and @2014) during Interval(0, 2) = false, want null"),
    ("includes read as not about a point",
     (EVAL, "\tcase ast.TimingIncludes:\n\t\treturn uncertainContained(right, left, op.Properly)\n", "\tcase ast.TimingIncludes:\n\t\treturn nil, false, nil\n"), HAND,
     "Interval[0, 2) includes (years between @2012 and @2014) = false, want null"),
    ("an open low bound read as closed",
     (EVAL, "if !iv.LowClosed || properly {", "if properly {"), GRID,
     "want"),
    ("an open high bound read as closed",
     (EVAL, "if !iv.HighClosed || properly {", "if properly {"), GRID,
     "want"),
    ("properly between read as between",
     (EVAL, "if !iv.LowClosed || properly {", "if !iv.LowClosed {"), HAND,
     "properly between 1 and 3 = true, want null"),
    ("membership of a list that answers false again",
     (EVAL, "return uncertainInList(x, c, properly), true, nil", "return fptypes.NewBoolean(false), true, nil"), HAND,
     "in { 1, 2 } = false, want null"),
    ("a list element that cannot coincide counted as unknown",
     (EVAL, "if !handled || res == nil {\n\t\t\tunknown = true", "if !handled || res == nil || true {\n\t\t\tunknown = true"), HAND,
     "in { 5, 6 } = null, want false"),
    ("the order phrases read as equality",
     (EVAL, "\tdefault:\n\t\tcmp = ast.OpGreater\n", "\tdefault:\n\t\tcmp = ast.OpGreaterOrEqual\n"), HAND,
     "after 1 = "),
    ("between that asks only about an uncertain operand, not an uncertain bound",
     (EVALUATOR, "if isUncertainty(operand) || isUncertainty(low) || isUncertainty(high) {", "if isUncertainty(operand) {"), HAND,
     "5 between (years between @2012 and @2014) and 10 = error"),
    ("an interval whose bound is an uncertainty not seen through ELM",
     (EVAL, "\t\treturn isUncertainty(iv.Low) || isUncertainty(iv.High)\n", "\t\treturn isUncertainty(iv.Low) && false\n"), HAND,
     "5 between (years between @2012 and @2014) and 10, through ELM = error"),
    ("an interval inside an interval with an uncertain bound read as a point",
     (EVAL, "\t\tif !pointOrUncertainty(x) {\n\t\t\treturn nil, false, nil\n\t\t}\n\t\tres, err := uncertainWithin", "\t\tif x == nil {\n\t\t\treturn nil, false, nil\n\t\t}\n\t\tres, err := uncertainWithin"),
     run("TestAnIntervalInAnIntervalWithAnUncertainBoundIsNotAPoint"), "= null, read as a point"),
    ("properly includes read as includes",
     (EVAL, "\tcase ast.TimingIncludes:\n\t\treturn uncertainContained(right, left, op.Properly)\n", "\tcase ast.TimingIncludes:\n\t\treturn uncertainContained(right, left, false)\n"), HAND,
     "properly includes (years between @2012 and @2014) = true, want null"),
    ("properly over a list of one element left unguarded",
     (EVAL, "if properly && len(items.Values) <= 1 {", "if false {"), HAND,
     "{ 1 } properly includes (years between @2012 and @2014) = null, want false"),
    ("duration in X of an interval back in UTC",
     (EVALUATOR, "return funcs.DurationBetweenAt(iv.Low, iv.High, n.Precision, e.evaluationFrame())", "return funcs.DurationBetweenAt(iv.Low, iv.High, n.Precision, nil)"), HAND,
     "duration in days of Interval[@2017-08-07T17:00, @2017-08-14T] = Interval[5, 6], want Interval[6, 7]"),
    # The comparison of two uncertainties.
    ("the extremes of two uncertainties swapped",
     (EVALUATOR, "lowCmp, lowErr := lowC.Compare(scalarHigh)", "lowCmp, lowErr := lowC.Compare(scalarLow)"), HAND,
     "(years between @2012 and @2014) > (years between @2011 and @2012) = true, want null"),
    # The frame of an incomplete DateTime.
    ("the range of a coarse DateTime back in UTC",
     (ADVANCED, "if _, placed := t.EffectiveOffset(); !placed && frame != nil {", "if false {"), OFFSETS,
     "= Interval[5, 6], want Interval[6, 7]"),
    ("the evaluation's offset never given to the range",
     (ADVANCED, "temporalRange(low, evaluation)", "temporalRange(low, nil)"), OFFSETS,
     "want Interval"),
    # Seconds and milliseconds.
    ("seconds and milliseconds a precision each again",
     (ADVANCED, "\tif index == 5 {\n\t\treturn 6\n\t}\n", "\tif false {\n\t\treturn 6\n\t}\n"), HAND,
     "seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0 = Interval[0, 1], want 1"),
    # The area and its classification.
    ("a deliberate divergence citing a plan that does not name it",
     ("testdata/snapshots/durations.divergences.tsv", "D46\tjs\tdeliberate\tInteger 2\tdeliberate: CQL 1.5.3 gives no rule for two operands both finer than the unit (its examples, Language Semantics and the Author's Guide, are all of one operand coarser than the unit, and the suite's `years between DateTime(2005) and DateTime(2010)` is [4, 5] with both at the year); this engine keeps its rule (an uncertainty only where an operand is not finer than the unit, as the Java engine reads it too), cql-execution reads every incomplete DateTime as a range and answers [1, 2]; the decision of the user on 31-10-2026 is in docs/plans/016-duration-uncertainty.md",
      "D46\tjs\tdeliberate\tInteger 2\tdeliberate: CQL 1.5.3 gives no rule for two operands both finer than the unit (its examples, Language Semantics and the Author's Guide, are all of one operand coarser than the unit, and the suite's `years between DateTime(2005) and DateTime(2010)` is [4, 5] with both at the year); this engine keeps its rule (an uncertainty only where an operand is not finer than the unit, as the Java engine reads it too), cql-execution reads every incomplete DateTime as a range and answers [1, 2]; the decision of the user on 31-10-2026 is in docs/plans/014-todate-toratio.md"),
     CONSISTENT, "D46"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"uncertainty", 1237, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "uncertainty.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", "durations uncertainty", "durations"),
     run("TestEveryAreaIsRunByTheReferences"), "scripts/references.sh does not run it"),
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
        print(f"BUILD FAILED {name}: a mutation that does not compile proves nothing\n{output[-1500:]}")
        survived += 1
    elif r.returncode == 0 or text not in output:
        print(f"SURVIVED     {name}: expected a failure saying {text!r}\n{output[-1500:]}")
        survived += 1
    else:
        print(f"caught       {name}")
sys.exit(1 if survived else 0)
PY
