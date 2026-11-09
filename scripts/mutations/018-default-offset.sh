#!/bin/sh
# Mutations of what plan 018 holds (docs/plans/018-date-to-datetime.md): a
# DateTime coarser than the hour is not placed at the evaluation offset for
# comparison (types.WithDefaultOffset, the only caller of fhirpath's method), and
# `timezoneoffset from` answers the offset of the evaluation request for a
# DateTime that has none, at every precision. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/018-default-offset.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-018.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, re, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-failfast", "-run", "^(" + "|".join(names) + ")$", package]


GUARD = "types/default_offset.go"
PRECISION = run("TestWithDefaultOffsetPlacesFromTheHour", package="./types")
MINMAX = run("TestMinMaxSortOfADayAndAnOffsetValueStayAsTheyWere")
SITES = run("TestEachPlaceOfDefaultOffsetPlacesFromTheHour")
STATIC = run("TestOnlyTheGuardCallsWithDefaultOffset")
TZ = run("TestTimezoneOffsetOfAnUnplacedDateTimeIsTheEvaluationOffset")
TZ_WRITTEN = run("TestTimezoneOffsetKeepsWhatTheValueStates")
DOORS = run("TestWhatIsNotADoor")

GUARD_LINE = "dt.Precision() < fptypes.DTHourPrecision"
TZ_OLD = '''		if _, placed := dt.EffectiveOffset(); !placed {
			_, seconds := e.ctx.EvaluationTimestamp.Zone()
			return fptypes.NewDecimalFromFloat(float64(seconds) / 3600.0), nil
		}'''

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("the guard also places below the hour",
     (GUARD, GUARD_LINE, "false"), MINMAX,
     "= false, want true"),
    ("the guard also places below the hour: by precision",
     (GUARD, GUARD_LINE, "false"), PRECISION,
     "placed true; want placed false"),
    ("the guard places only from the minute",
     (GUARD, GUARD_LINE, "dt.Precision() < fptypes.DTMinutePrecision"), PRECISION,
     "WithDefaultOffset(2020-03-01T10, "),
    ("a literal calls fhirpath's WithDefaultOffset directly again",
     ("eval/evaluator.go", "return cqltypes.WithDefaultOffset(dt, time.Duration(seconds)*time.Second), nil",
      "return dt.WithDefaultOffset(time.Duration(seconds) * time.Second), nil"), STATIC,
     "eval/evaluator.go"),
    ("situate calls fhirpath's WithDefaultOffset directly again",
     ("eval/planned_type.go", "return cqltypes.WithDefaultOffset(dt, time.Duration(seconds)*time.Second)",
      "_ = cqltypes.WithDefaultOffset\n\treturn dt.WithDefaultOffset(time.Duration(seconds) * time.Second)"), STATIC,
     "eval/planned_type.go"),
    ("timezoneoffset back to null for an unplaced DateTime",
     ("eval/evaluator.go", TZ_OLD, TZ_OLD.replace("return fptypes.NewDecimalFromFloat(float64(seconds) / 3600.0), nil", "_ = seconds\n\t\t\treturn nil, nil")), TZ,
     "= <nil>, want"),
    ("timezoneoffset back to null: the extraction case of the doors",
     ("eval/evaluator.go", TZ_OLD, TZ_OLD.replace("return fptypes.NewDecimalFromFloat(float64(seconds) / 3600.0), nil", "_ = seconds\n\t\t\treturn nil, nil")), DOORS,
     "a DateTime no finer than the day answers null"),
    ("timezoneoffset ignores a written offset",
     ("eval/evaluator.go", "if _, placed := dt.EffectiveOffset(); !placed {\n\t\t\t_, seconds := e.ctx.EvaluationTimestamp.Zone()\n\t\t\treturn fptypes.NewDecimalFromFloat",
      "if _, placed := dt.EffectiveOffset(); !placed || true {\n\t\t\t_, seconds := e.ctx.EvaluationTimestamp.Zone()\n\t\t\treturn fptypes.NewDecimalFromFloat"), TZ_WRITTEN,
     "want 7"),
]

# A deliberate verdict of the area names its row in plan 018 (TestTheDivergencesAreConsistent).
plan = open(os.path.join(repo, "docs/plans/018-date-to-datetime.md")).read()
divergences = [l.split("\t") for l in open(os.path.join(repo, "testdata/snapshots/mixeddates.divergences.tsv")).read().split("\n")
               if l and not l.startswith("#")]
named_once = [d[0] for d in divergences if d[2] == "deliberate" and "docs/plans/018-date-to-datetime.md" in d[4]
              and len(re.findall(r"(?<![\w-])" + d[0] + r"(?![\w-])", plan)) == 1 and f" {d[0]}," in plan]
if not named_once:
    print("NO DELIBERATE ROW of plan 018 to mutate")
    sys.exit(2)
mutations.append(
    ("a deliberate row that cites a plan that does not name it",
     ("docs/plans/018-date-to-datetime.md", f" {named_once[0]},", " DX,"), run("TestTheDivergencesAreConsistent"),
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
