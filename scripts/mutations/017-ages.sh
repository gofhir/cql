#!/bin/sh
# Mutations of what plan 017 holds (docs/plans/017-age-duration.md): an age is a
# duration in its unit, in the evaluation's frame; a Date beside a DateTime, or in
# a unit finer than a day, is converted with ToDateTime; the seven units; the
# overloads without an asOf (a Date against Today(), a DateTime against Now());
# the ELM importer's CalculateAge and CalculateAgeAt; the fixed timestamp of
# evalImported; and the area that holds the cases. Each is applied to a copy of
# the repository; the command it names has to fail, and its output has to say why.
# The mutations that need the references of the area (a deliberate row citing a
# plan that does not name it) are not here.
# Usage: sh scripts/mutations/017-ages.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-017.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


HAND = run("TestAgesByHand")
GRID = run("TestAnAgeIsAnsweredByItsOracle")
PATIENT = run("TestTheAgeOfAPatient")
ELM = run("TestAgesFromTheReferenceELM")
CLINICAL = "funcs/clinical.go"
EVALUATOR = "eval/evaluator.go"
IMPORTER = "elm/importer.go"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The age as a duration.
    ("the age back to a duration that is never an uncertainty",
     (CLINICAL, "return DurationBetweenAt(birthDate, asOf, unit, evaluation)", "return durationBetweenExact(birthDate, asOf, unit)"), HAND,
     "CalculateAgeInWeeksAt(@2019-01, @2019-03-01) = "),
    ("the age measured in UTC and not in the evaluation's frame",
     (CLINICAL, "return DurationBetweenAt(birthDate, asOf, unit, evaluation)", "return DurationBetweenAt(birthDate, asOf, unit, nil)"), HAND,
     "CalculateAgeInDaysAt(@2019-01-01T, @2019-01-02T09:00:00.000Z) = Interval[0, 1], want 0"),
    ("the evaluation's frame never given to the age",
     (EVALUATOR, "return funcs.CalculateAgeIn(unit, birthDate, asOf, e.evaluationFrame())", "return funcs.CalculateAgeIn(unit, birthDate, asOf, nil)"), HAND,
     "CalculateAgeInDaysAt(@2019-01-01T, @2019-01-02T09:00:00.000Z) = Interval[0, 1], want 0"),
    # The Date and the DateTime.
    ("two Dates left as Dates in a unit finer than a day",
     (CLINICAL, "if birthIsDate && asOfIsDate && !AgeNeedsDateTime(unit) {", "if birthIsDate && asOfIsDate {"), HAND,
     "CalculateAgeInHoursAt(@2019-01-01, @2019-01-02) = 24, want Interval[0, 47]"),
    ("the Date beside a DateTime not converted when it is the reference",
     (CLINICAL, "reference, err = ToDateTime(asOf)", "reference, err = asOf, error(nil)"), GRID,
     "want"),
    ("the Date beside a DateTime not converted when it is the birth date",
     (CLINICAL, "birth, err = ToDateTime(birthDate)", "birth, err = birthDate, error(nil)"), GRID,
     "want"),
    ("the units that need a DateTime reduced to hours",
     (CLINICAL, 'case "hours", "minutes", "seconds":', 'case "hours":'), HAND,
     "CalculateAgeInMinutesAt(@2019-01-01, @2019-01-02) = "),
    # The overloads without an asOf.
    ("Today() taken as Now() for a Date without an asOf",
     (EVALUATOR, "if asOf, err = funcs.TodayAt(e.ctx.EvaluationTimestamp); err != nil {", "if asOf, err = e.evaluationNow(), error(nil); err != nil {"), HAND,
     "CalculateAgeInDays(@2025-12-31) = "),
    ("the Date overload of a unit up to days never taken",
     (EVALUATOR, "|| calculate && !funcs.AgeNeedsDateTime(unit)) {", ") {"), HAND,
     "CalculateAgeInDays(@2025-12-31) = "),
    ("Now() taken as Today() for the patient's days",
     (EVALUATOR, "\t\t} else {\n\t\t\tasOf = e.evaluationNow()\n\t\t}\n", "\t\t} else {\n\t\t\tasOf, _ = funcs.TodayAt(e.ctx.EvaluationTimestamp)\n\t\t}\n"), PATIENT,
     "AgeInDays() = 16802, want Interval[16801, 16802]"),
    ("the patient's days read as a Date against Today()",
     (EVALUATOR, "if isDate && (unit == \"years\" || unit == \"months\" ||", "if isDate && (true ||"), PATIENT,
     "AgeInDays() = 16802, want Interval[16801, 16802]"),
    # The units.
    ("minutes read as seconds",
     (EVALUATOR, "unit := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(name, \"calculate\"), \"agein\"), \"at\")\n", "unit := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(name, \"calculate\"), \"agein\"), \"at\")\n\tif unit == \"minutes\" {\n\t\tunit = \"seconds\"\n\t}\n"), HAND,
     "CalculateAgeInMinutesAt(@2019-01-01T10:00:00, @2019-01-02T09:00:00) = "),
    ("weeks read as days",
     (EVALUATOR, "unit := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(name, \"calculate\"), \"agein\"), \"at\")\n", "unit := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(name, \"calculate\"), \"agein\"), \"at\")\n\tif unit == \"weeks\" {\n\t\tunit = \"days\"\n\t}\n"), HAND,
     "CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07) = "),
    ("the hours at left unregistered",
     (EVALUATOR, '"calculateageinhoursat", "calculateageinminutesat"', '"calculateageinhoursatx", "calculateageinminutesat"'), HAND,
     "CalculateAgeInHoursAt(@2019-01-01T10:00, @2019-01-02T09:00) = "),
    # The ELM importer.
    ("the importer without CalculateAge and CalculateAgeAt",
     (IMPORTER, 'case "CalculateAge", "CalculateAgeAt":', 'case "CalculateAgeNever":'), ELM,
     "AgeInYears from ELM = null, want 46"),
    ("the importer reading the precision as a day",
     (IMPORTER, 'name := "CalculateAgeIn" + node.Precision + "s"', 'name := "CalculateAgeInDays"'), ELM,
     "AgeInYears from ELM = "),
    ("the importer reading a path as one member",
     (IMPORTER, 'strings.Split(node.Path, ".")', '[]string{node.Path}'), ELM,
     "AgeInYears from ELM = null, want 46"),
    ("the importer without ToDateTime",
     (IMPORTER, 'case "ToDateTime":', 'case "ToDateTimeNever":'), ELM,
     "AgeInDays from ELM = null"),
    ("Today() of an age in UTC rather than the evaluation's offset",
     ("eval/evaluator.go", "asOf, err = funcs.TodayAt(e.ctx.EvaluationTimestamp)", "asOf, err = funcs.TodayAt(e.ctx.EvaluationTimestamp.UTC())"),
     run("TestTodayIsTheEvaluationsDate"), "CalculateAgeInDays(@2025-12-31) = 0"),
    # evalImported.
    ("evalImported at the real clock",
     ("engine_durations_test.go", "\tctx.EvaluationTimestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone(\"\", -12*3600))\n", ""), HAND,
     "imported, CalculateAgeInYears(@1965-06-01) = "),
    # The area.
    ("a deliberate divergence citing a plan that does not name it",
     ("testdata/snapshots/ages.divergences.tsv", "and docs/plans/017-age-duration.md names this row; cql-execution agrees with this engine, the Java engine ignores",
      "and docs/plans/014-todate-toratio.md names this row; cql-execution agrees with this engine, the Java engine ignores"),
     run("TestTheDivergencesAreConsistent"), "which does not name D40"),
    ("an age of the area away from its oracle",
     ("testdata/snapshots/ages.answers.tsv", "D103\tUncertainty[Integer 241, Integer 253]", "D103\tUncertainty[Integer 242, Integer 253]"),
     run("TestTheAgesAreaIsItsOracle"), "D103"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"ages", 462, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "ages.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", "uncertainty ages", "uncertainty"),
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
