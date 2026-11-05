#!/bin/sh
# Mutations of what plan 023 holds (docs/plans/023-interval-start-end.md):
# `duration in X of I` and `difference in X of I` measure between `start of I`
# and `end of I`, and those read a closed null boundary as the minimum or maximum
# of a temporal point type, an open null as null, and a step past the calendar or
# the clock as null. Each is applied to a copy of the repository; the command it
# names has to fail, and its output has to say why. The last two edit the oracle
# test and the answers of the area.
# Not here, until the references exist: a divergence of the area left
# unclassified (needs intervalbounds.divergences.tsv).
# Usage: sh scripts/mutations/023-interval-start-end.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-023.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-failfast", "-run", "^(" + "|".join(names) + ")$", package]


HAND = run("TestIntervalBoundsByHand")
ORACLE = run("TestTheIntervalBoundsAreTheirOracle")
EVAL = "eval/evaluator.go"
PRECISION = "eval/interval_precision.go"
LOW = "if iv.Low == nil && iv.LowClosed {"
HIGH = "if iv.High == nil && iv.HighClosed {"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("duration of reads Low and High again",
     (EVAL, "return funcs.DurationBetweenAt(start, end, n.Precision, e.evaluationFrame())", "return funcs.DurationBetweenAt(iv.Low, iv.High, n.Precision, e.evaluationFrame())"), HAND,
     "duration in days of Interval(@2012-01-01, @2012-01-05) = 4, want 2"),
    ("difference of reads Low and High again",
     (EVAL, "return funcs.DifferenceBetweenAt(start, end, n.Precision, e.evaluationFrame())", "return funcs.DifferenceBetweenAt(iv.Low, iv.High, n.Precision, e.evaluationFrame())"), HAND,
     "difference in days of Interval(@2012-01-01, @2012-01-05) = 4, want 2"),
    ("a closed null start is null again",
     (PRECISION, LOW, "if false && iv.Low == nil && iv.LowClosed {"), HAND,
     "duration in days of Interval[null, @2012-01-01] = null, want 734502"),
    ("a closed null end is null again",
     (PRECISION, HIGH, "if false && iv.High == nil && iv.HighClosed {"), HAND,
     "duration in days of Interval[@2012-01-01, null] = null, want 2917556"),
    ("the minimum and the maximum swapped at the start",
     (PRECISION, "return extremeLike(iv.High, false), nil", "return extremeLike(iv.High, true), nil"), HAND,
     "duration in days of Interval[null, @2012-01-01] = -"),
    ("the maximum DateTime at the evaluation offset instead of UTC",
     (PRECISION, 'fptypes.NewDateTime("9999-12-31T23:59:59.999Z")', 'fptypes.NewDateTime("9999-12-31T23:59:59.999-03:00")'), HAND,
     "= 23, want 20"),
    ("the minimum DateTime at the evaluation offset instead of UTC",
     (PRECISION, 'fptypes.NewDateTime("0001-01-01T00:00:00.000Z")', 'fptypes.NewDateTime("0001-01-01T00:00:00.000-03:00")'), HAND,
     "want 0001-01-01T00:00:00.000Z"),
    ("start of stops using startOf",
     (EVAL, "v, err := startOf(iv)", "v, err := iv.Start()"), HAND,
     "start of Interval[null, @2012-01-01] = null, want 0001-01-01"),
    ("end of stops using endOf",
     (EVAL, "v, err := endOf(iv)", "v, err := iv.End()"), HAND,
     "end of Interval[@2012-01-01, null] = null, want 9999-12-31"),
    ("no step check on start of: the successor of 9999-12-31",
     (EVAL, "v, err := startOf(iv)\n\t\t\treturn e.stepWithinItsType(n, v, err)", "v, err := startOf(iv)\n\t\t\treturn v, err"), HAND,
     "start of Interval(@9999-12-31, null] = error"),
    ("no step check on end of: the predecessor of 0001-01-01",
     (EVAL, "v, err := endOf(iv)\n\t\t\treturn e.stepWithinItsType(n, v, err)", "v, err := endOf(iv)\n\t\t\treturn v, err"), HAND,
     "end of Interval[null, @0001-01-01) = error"),
    ("no step check on the start of a duration",
     (PRECISION, "if start, err = e.stepWithinItsType(node, start, err); err != nil || start == nil {", "if err != nil || start == nil {"), HAND,
     "duration in days of Interval(@9999-12-31, null] = error"),
    ("no step check on the end of a duration: the predecessor of 0001-01-01",
     (PRECISION, "if end, err = e.stepWithinItsType(node, end, err); err != nil || end == nil {", "if err != nil || end == nil {"), HAND,
     "duration in days of Interval[null, @0001-01-01) = error"),
    ("an open null start is read as the minimum",
     (PRECISION, LOW, "if iv.Low == nil {"), HAND,
     "duration in days of Interval(null, @2012-01-01] = 734502, want null"),
    ("an open null end is read as the maximum",
     (PRECISION, HIGH, "if iv.High == nil {"), HAND,
     "duration in days of Interval[@2012-01-01, null) = 2917556, want null"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"intervalbounds", 597, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "intervalbounds.cql is not in snapshotAreas"),
    ("an answer of the area moved away from the oracle",
     ("testdata/snapshots/intervalbounds.answers.tsv", "\nD100\tInteger 9998\n", "\nD100\tInteger 9999\n"), ORACLE,
     "= 9999, the oracle gives 9998"),
    ("a classified divergence of the area left unclassified",
     ("testdata/snapshots/intervalbounds.divergences.tsv", "\nD85\tjava\treference defect\t", "\nD85\tjava\tunclassified\t"),
     run("TestThePendingVerdictsDoNotGrow"), "has no verdict and is not in pending-verdicts.tsv"),
    ("an answer of the oracle moved away from the engine",
     ("testdata/intervalbounds/oracle.tsv", "\nD100\tInteger 9998\n", "\nD100\tInteger 9997\n"), ORACLE,
     "the oracle gives 9997"),
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
