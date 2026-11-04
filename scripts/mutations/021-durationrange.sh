#!/bin/sh
# Mutations of what plan 021 holds (docs/plans/021-duration-integer-range.md): a
# duration, a difference and an age are measured over the whole calendar, from
# the first instant of the year 1 (not "no value") to 9999, and a span of more
# than 292 years (the range of a Go time.Duration) is counted whole. Each is
# applied to a copy of the repository; the command it names has to fail, and its
# output has to say why.
# Usage: sh scripts/mutations/021-durationrange.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-021.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


HAND = run("TestDurationsOverTheCalendarByHand")
TEMPORAL = "funcs/temporal.go"
ADVANCED = "funcs/datetime_advanced.go"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("the zero time read as no value again, in the conversion every XBetween uses",
     (TEMPORAL, "\tif !lok || !hok {\n\t\treturn 0, 0, false", "\tif !lok || !hok || tl.IsZero() || th.IsZero() {\n\t\treturn 0, 0, false"), HAND,
     "days between @0001-01-01 and @0001-01-05 = null, want 4"),
    ("hours measured with time.Time.Sub again",
     (TEMPORAL, "\treturn wholeSeconds(low, high, 3600)",
      "\ttl, _ := toGoTime(low)\n\tth, _ := toGoTime(high)\n\treturn fptypes.NewInteger(int64(th.Sub(tl).Hours())), nil"), HAND,
     "hours between @1700-01-01T00:00:00.000 and @2020-01-01T00:00:00.000 = 2562047, want 2805048"),
    ("hours measured with Sub, the other way round",
     (TEMPORAL, "\treturn wholeSeconds(low, high, 3600)",
      "\ttl, _ := toGoTime(low)\n\tth, _ := toGoTime(high)\n\treturn fptypes.NewInteger(int64(th.Sub(tl).Hours())), nil"), HAND,
     "hours between @2020-01-01T00:00:00.000 and @1700-01-01T00:00:00.000 = -2562047, want -2805048"),
    ("the days of a Date measured with Sub again",
     (TEMPORAL, "\treturn wholeSeconds(low, high, 86400)",
      "\ttl, _ := toGoTime(low)\n\tth, _ := toGoTime(high)\n\treturn fptypes.NewInteger(int64(th.Sub(tl).Hours() / 24)), nil"), HAND,
     "days between @0001-01-02 and @9999-12-31 = 106751, want 3652057"),
    ("the minutes measured with Sub again",
     (TEMPORAL, "\treturn wholeSeconds(low, high, 60)",
      "\ttl, _ := toGoTime(low)\n\tth, _ := toGoTime(high)\n\treturn fptypes.NewInteger(int64(th.Sub(tl).Minutes())), nil"), HAND,
     "minutes between @1727-01-01T00:00:00.000 and @2020-01-01T00:00:00.000 = 153722867, want 154103040"),
    ("the uncertainty built from a zero time that is the first instant",
     (ADVANCED, "\tminLowDT := fptypes.NewDateTimeFromTime(minLow)\n",
      "\tif maxLow.IsZero() {\n\t\treturn nil, nil\n\t}\n\tminLowDT := fptypes.NewDateTimeFromTime(minLow)\n"), HAND,
     "minutes between @0001-01-01T00:00Z and @0001-01-01T05:00:00.000Z = null, want Interval[299, 300]"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"durationrange", 1675, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "durationrange.cql is not in snapshotAreas"),
    ("a negative span's leftover part of a second not borrowed",
     ("funcs/temporal.go", "\tcase seconds < 0 && nanos > 0:\n", "\tcase false:\n"), run("TestDurationsOverTheCalendarByHand"),
     "seconds between @2012-01-01T00:00:01.000Z and @2012-01-01T00:00:00.500Z = -1, want 0"),
    ("an answer of the area moved away from the oracle",
     ("testdata/snapshots/durationrange.answers.tsv", "\nD3\tInteger 0\n", "\nD3\tInteger 1\n"), run("TestTheDurationRangeAreaIsItsOracle"),
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
