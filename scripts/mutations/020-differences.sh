#!/bin/sh
# Mutations of what plan 020 holds (docs/plans/020-difference-uncertainty.md):
# `difference in X between` counts the boundaries of X crossed (hours and finer
# in the evaluation's frame, each instant cut down to the unit; Sundays for
# weeks) and is an uncertainty where an operand is coarser than X, the second and
# the millisecond being two precisions. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# The mutations that need the references or the answers of the area (an answer
# away from its oracle, a deliberate row citing a plan that does not name it) are
# not here: they need files that are generated, not written.
# Usage: sh scripts/mutations/020-differences.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-020.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


HAND = run("TestDifferencesByHand")
OFFSETS = run("TestDifferencesAtOtherOffsets")
FUNC = "funcs/datetime_advanced.go"
EVALUATOR = "eval/evaluator.go"
IMPORTER = "elm/importer.go"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("hours measured as the time elapsed, not the boundaries crossed",
     (FUNC, "return (dayNumber(b)*24 + int64(b.Hour())) - (dayNumber(a)*24 + int64(a.Hour()))", "return int64(b.Sub(a).Hours())"), HAND,
     "difference in hours between @2012-01-31T23:59:59.999 and @2012-02-01T00:00:00.000 = 0, want 1"),
    ("minutes measured as the time elapsed",
     (FUNC, "return (dayNumber(b)*1440 + int64(b.Hour()*60+b.Minute())) - (dayNumber(a)*1440 + int64(a.Hour()*60+a.Minute()))", "return int64(b.Sub(a).Minutes())"), HAND,
     "difference in minutes between @T10:59:30.000 and @T11:01:10.000 = 1, want 2"),
    ("weeks as days / 7",
     (FUNC, "return (sundayOf(b) - sundayOf(a)) / 7", "return (dayNumber(b) - dayNumber(a)) / 7"), HAND,
     "difference in weeks between @2012-01-07 and @2012-01-08 = 0, want 1"),
    ("weeks that start on Monday",
     (FUNC, "return dayNumber(t) - int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Weekday())",
      "return dayNumber(t) - int64((time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Weekday()+6)%7)"), HAND,
     "difference in weeks between @2012-01-07 and @2012-01-08 = 0, want 1"),
    ("an uncertainty never",
     (FUNC, "if differencePrecision(low) < unit || differencePrecision(high) < unit {", "if false {"), HAND,
     "difference in months between @2012-01-02 and @2012 = 0, want Interval[0, 11]"),
    ("the second and the millisecond as one precision",
     (FUNC, "\tcase fptypes.DateTime:\n\t\treturn int(t.Precision())\n\tcase fptypes.Time:", "\tcase fptypes.DateTime:\n\t\treturn combinedSecondsAndMilliseconds(int(t.Precision()))\n\tcase fptypes.Time:"), HAND,
     "difference in milliseconds between @2012-01-31T23:59:59 and @2012-02-01T00:00:00.000 = 1000, want Interval[1, 1000]"),
    ("the uncertainty's bounds swapped",
     (FUNC, "return cqltypes.NewUncertainty(fptypes.NewInteger(smallest), fptypes.NewInteger(largest)), nil", "return cqltypes.NewUncertainty(fptypes.NewInteger(largest), fptypes.NewInteger(smallest)), nil"), HAND,
     "difference in months between @2012-01-02 and @2012 = Interval[11, 0], want Interval[0, 11]"),
    ("the low bound from the latest of both",
     (FUNC, "smallest := boundariesCrossed(lowLatest, highEarliest,", "smallest := boundariesCrossed(lowLatest, highLatest,"), HAND,
     "difference in hours between @2012-02-01T00:00:00.000 and @2012-01-31T = "),
    ("the evaluation's frame never given by a difference",
     (EVALUATOR, "return funcs.DifferenceBetweenAt(low, high, n.Precision, e.evaluationFrame())", "return funcs.DifferenceBetweenAt(low, high, n.Precision, nil)"), HAND,
     "difference in hours between @2012-01-31T and @2012-02-01T00:00:00.000 = Interval[4, 27], want Interval[1, 24]"),
    ("the evaluation's frame never given by a difference of an interval",
     (EVALUATOR, "return funcs.DifferenceBetweenAt(iv.Low, iv.High, n.Precision, e.evaluationFrame())", "return funcs.DifferenceBetweenAt(iv.Low, iv.High, n.Precision, nil)"), HAND,
     "difference in hours of Interval[@2012-01-31T, @2012-02-01T00:00:00.000] = Interval[4, 27], want Interval[1, 24]"),
    ("hours counted in UTC and not in the evaluation's frame (at +05:45)",
     (FUNC, "\t\ta, b = a.In(frame), b.In(frame)\n", "\t\ta, b = a.UTC(), b.UTC()\n"), OFFSETS,
     "difference in"),
    ("a Time moved into the evaluation's frame",
     (FUNC, "\tif !isTime {\n\t\tif frame == nil {", "\tif true {\n\t\tif frame == nil {"), OFFSETS,
     "difference in"),
    ("the ELM importer without DifferenceBetween",
     (IMPORTER, 'case "DifferenceBetween":', 'case "DifferenceBetweenNever":'), HAND,
     "imported, difference in"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"differences", 776, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "differences.cql is not in snapshotAreas"),
    ("an answer of the area moved away from the oracle",
     ("testdata/snapshots/differences.answers.tsv", "D2\tInteger 0\n", "D2\tInteger 1\n"), run("TestTheDifferencesAreaIsItsOracle"),
     "= 1, the oracle gives 0"),
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
