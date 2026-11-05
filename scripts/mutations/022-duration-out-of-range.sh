#!/bin/sh
# Mutations of what plan 022 holds (docs/plans/022-duration-out-of-range.md): a
# duration, a difference and an age are Integers, and a count past 2^31-1 or
# below -2^31 is null, an uncertainty with a bound past them whole; and the
# arithmetic of plan 013 knows a duration for an Integer. Each is applied to a
# copy of the repository; the command it names has to fail, and its output has
# to say why. The last two edit the oracle test and the answers of the area, and
# pass the baseline only once the references are regenerated for the area.
# Usage: sh scripts/mutations/022-duration-out-of-range.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-022.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


EDGE = run("TestDurationsAtTheIntegerEdge")
CERTAIN = run("TestADurationIsAnIntegerForCertain")
ORACLE = run("TestTheDurationRangeAreaIsItsOracle")
ADVANCED = "funcs/datetime_advanced.go"
TESTS = "engine_durationrange_test.go"
LAST_MS = "and @2012-01-25T20:31:23.648 = "
CROSSING = "and @2012-01-25T20:31Z = Interval[2147460000, 2147519999], want null"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("the upper edge at 2^31 instead of 2^31-1",
     (ADVANCED, "i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32", "i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32+1"), EDGE,
     LAST_MS + "2147483648, want null"),
    ("the lower edge at -2^31+1 instead of -2^31",
     (ADVANCED, "i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32", "i.Value() > math.MinInt32 && i.Value() <= math.MaxInt32"), EDGE,
     "= null, want -2147483648"),
    ("the lower edge at -2^31-1 instead of -2^31",
     (ADVANCED, "i.Value() >= math.MinInt32 && i.Value() <= math.MaxInt32", "i.Value() >= math.MinInt32-1 && i.Value() <= math.MaxInt32"), EDGE,
     "= -2147483649, want null"),
    ("no check in Duration",
     (ADVANCED, "v, err := durationBetweenAt(low, high, precision, evaluation)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn inIntegerRange(v), nil",
      "return durationBetweenAt(low, high, precision, evaluation)"), EDGE,
     "milliseconds between @2012-01-01T00:00:00.000 " + LAST_MS + "2147483648, want null"),
    ("no check in Difference",
     (ADVANCED, "v, err := differenceBetweenAt(low, high, precision, evaluation)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn inIntegerRange(v), nil",
      "return differenceBetweenAt(low, high, precision, evaluation)"), EDGE,
     "difference in milliseconds between @2012-01-01T00:00:00.000 " + LAST_MS + "2147483648, want null"),
    ("no check on an uncertainty",
     (ADVANCED, "\tcase cqltypes.Uncertainty:\n\t\tlow, okLow", "\tcase cqltypes.Uncertainty:\n\t\treturn v\n\t\tlow, okLow"), EDGE,
     CROSSING),
    ("the check on the low bound of an uncertainty only",
     (ADVANCED, "(okLow && !fitsInteger(low)) || (okHigh && !fitsInteger(high))", "(okLow && !fitsInteger(low)) || (okHigh && false && !fitsInteger(high))"), EDGE,
     CROSSING),
    ("the check on the high bound of an uncertainty only",
     (ADVANCED, "(okLow && !fitsInteger(low)) || (okHigh && !fitsInteger(high))", "(okLow && false && !fitsInteger(low)) || (okHigh && !fitsInteger(high))"), EDGE,
     "Interval[-2147519999, -2147460000], want null"),
    ("an age around the check",
     ("funcs/clinical.go", "return DurationBetweenAt(birthDate, asOf, unit, evaluation)", "return durationBetweenAt(birthDate, asOf, unit, evaluation)"), EDGE,
     "CalculateAgeInSecondsAt(@2000-01-01T00:00:00.000Z, @2068-01-19T03:14:08.000Z) = 2147483648, want null"),
    ("the count wrapped to 32 bits instead of null",
     (ADVANCED, "\t\tif !fitsInteger(x) {\n\t\t\treturn nil\n\t\t}", "\t\treturn fptypes.NewInteger(int64(int32(x.Value())))"), EDGE,
     LAST_MS + "-2147483648, want null"),
    ("a duration out of certainInteger",
     ("eval/integer_range.go", "case *ast.DurationBetween, *ast.DifferenceBetween, *ast.DurationOf, *ast.DifferenceOf:", "case *ast.DifferenceBetween, *ast.DurationOf, *ast.DifferenceOf:"), CERTAIN,
     "= 2147483648, want null"),
    ("a difference of an interval out of certainInteger",
     ("eval/integer_range.go", "case *ast.DurationBetween, *ast.DifferenceBetween, *ast.DurationOf, *ast.DifferenceOf:", "case *ast.DurationBetween, *ast.DifferenceBetween, *ast.DurationOf:"), CERTAIN,
     "(difference in milliseconds of Interval[@2012-01-01T00:00:00.000, @2012-01-25T20:31:23.647]) + 1 = 2147483648, want null"),
    ("an age out of certainInteger",
     ("eval/integer_range.go", "namedIn(n.Name, fixedInteger) || namedIn(n.Name, ageFunctions)", "namedIn(n.Name, fixedInteger)"), CERTAIN,
     "(CalculateAgeInSeconds(@1957-12-13T23:45:53.000Z)) + 1 = 2147483648, want null"),
    ("an uncertainty out of pastInteger",
     ("eval/integer_range.go", "\tcase cqltypes.Uncertainty:\n\t\treturn pastInteger32(x.Low) || pastInteger32(x.High)\n", ""), CERTAIN,
     "+ 100000 = Interval[2147440000, 2147499999], want null"),
    ("only the high bound of an uncertainty checked",
     ("eval/integer_range.go", "\t\treturn pastInteger32(x.Low) || pastInteger32(x.High)\n", "\t\treturn pastInteger32(x.High)\n"), CERTAIN,
     "- 1 = Interval[-2147483649, -2147423650], want null"),
    ("the negation of an uncertainty around withinInteger",
     ("eval/evaluator.go", "\t\t\tv, err := negateUncertainty(u)\n\t\t\treturn e.withinInteger(n, v, err)", "\t\t\treturn negateUncertainty(u)"), CERTAIN,
     "= Interval[2147423649, 2147483648], want null"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"durationrange", 1948, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "durationrange.cql is not in snapshotAreas"),
    ("the oracle without the edge: 2^31 kept as a number",
     (TESTS, "case n >= limit || n < -limit:", "case n > limit || n < -limit:"), ORACLE,
     "the oracle gives 2147483648"),
    ("an answer of the area moved away from the oracle",
     ("testdata/snapshots/durationrange.answers.tsv", "\nD3\tInteger 0\n", "\nD3\tInteger 1\n"), ORACLE,
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
