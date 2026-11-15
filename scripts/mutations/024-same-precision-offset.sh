#!/bin/sh
# Mutations of what plan 024 holds (docs/plans/024-same-precision-offset.md): a
# comparison at the hour or finer reads two DateTimes in what they wrote when
# their offsets are the same, and both in the offset of the evaluation request
# when they differ, in every phrase that compares at a precision; and the area
# `fractionaloffset` is evaluated at +05:30 by the references and by this engine.
# Each is applied to a copy of the repository; the command it names has to fail,
# and its output has to say why.
# Usage: sh scripts/mutations/024-same-precision-offset.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-024.XXXXXX")
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
IP = "eval/interval_precision.go"
PHRASES = run("TestEveryPhraseWithAPrecisionReadsTheHourInTheEvaluationOffset")
PROBES = run("TestAHourIsCutInTheEvaluationOffset")
SAME_WRITTEN = run("TestTheSameWrittenOffsetIsReadAsWritten")
DIFFERENT = run("TestDifferentOffsetsAreReadInTheEvaluationOffset")
ORACLE = run("TestTheFractionalOffsetAreaIsItsOracle")
AT_OFFSET = run("TestEachAreaIsEvaluatedAtItsOffset")
SNAPSHOT = run("TestAnswersMatchTheSnapshots")
REFERENCES = run("TestAnswersMatchTheReferences")
SCRIPT = run("TestTheReferencesScriptTellsTheHarnessTheOffsetOfEachArea")
HARNESS = run("TestTheHarnessReadsTheEvaluationOffset")
UNPLACED = run("TestAnUnplacedDateTimeIsAtTheOffsetOfTheEvaluation", package="./eval")
WINDOW = run("TestTheWindowOfAnUnplacedDateTimeIsReadInTheEvaluationOffset", package="./eval")
COMPARE = run("TestTheEngineIsAtTheOffsetOfTheArea", package="./internal/referenceharness/differential/compare")

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The reading itself.
    ("the reading is back at UTC",
     (EV, "case differentOffsets(a, b, frame):\n\t\treturn frame", "case differentOffsets(a, b, frame):\n\t\treturn time.UTC"), PROBES,
     "D21 at"),
    ("the reading is at UTC: the phrases",
     (EV, "case differentOffsets(a, b, frame):\n\t\treturn frame", "case differentOffsets(a, b, frame):\n\t\treturn time.UTC"), PHRASES,
     "want"),
    ("the reading normalizes even where the offsets are the same",
     (EV, "case differentOffsets(a, b, frame):\n\t\treturn frame", "case true:\n\t\treturn frame"), SAME_WRITTEN,
     "false"),
    ("the reading is at the offset of the left operand",
     (EV, "case differentOffsets(a, b, frame):\n\t\treturn frame",
      "case differentOffsets(a, b, frame):\n\t\treturn time.FixedZone(\"\", effectiveOffsetMinutes(a, frame)*60)"), DIFFERENT,
     "want"),
    ("a value with no offset is read in the evaluation's offset by the readers of components",
     (EV, "\tcase unplaced(a) || unplaced(b):", "\tcase false:"),
     run("TestAnUnplacedValueInAnIntervalIsReadAsBefore"),
     "want true, as before plan 024"),
    # Each caller of the frame.
    ("the point phrases pass UTC",
     (EV, "\tframe := e.evaluationFrame()\n\n\tswitch op.Kind {", "\tframe := time.UTC\n\n\tswitch op.Kind {"), PHRASES,
     "want"),
    ("an interval `same as` at a precision passes UTC",
     (EV, "return temporalSameAs(e.evaluationFrame(), a, b, precision)", "return temporalSameAs(time.UTC, a, b, precision)"), PHRASES,
     "same hour as Interval"),
    ("the timing offset phrase passes UTC",
     (EV, "return temporalCompareAtPrecision(e.evaluationFrame(), a, b, precision)", "return temporalCompareAtPrecision(time.UTC, a, b, precision)"), PHRASES,
     "or less before hour of"),
    ("`in` at a precision passes UTC",
     (EV, "membershipAtPrecision(e.evaluationFrame(), iv, point, n.Precision)", "membershipAtPrecision(time.UTC, iv, point, n.Precision)"), PHRASES,
     "in hour of"),
    ("`starts … during` at a precision passes UTC",
     (EV, "membershipAtPrecision(e.evaluationFrame(), rightIv, point, n.Operator.Precision)", "membershipAtPrecision(time.UTC, rightIv, point, n.Operator.Precision)"), PHRASES,
     "starts during hour of"),
    ("`includes` at a precision passes UTC",
     (EV, "return intervalIncludesAtPrecision(e.evaluationFrame(), leftIv, rightIv, n.Operator.Precision, n.Operator.Properly)",
      "return intervalIncludesAtPrecision(time.UTC, leftIv, rightIv, n.Operator.Precision, n.Operator.Properly)"), PHRASES,
     "includes hour of"),
    ("`during` at a precision passes UTC",
     (EV, "return intervalIncludesAtPrecision(e.evaluationFrame(), rightIv, leftIv, n.Operator.Precision, n.Operator.Properly)",
      "return intervalIncludesAtPrecision(time.UTC, rightIv, leftIv, n.Operator.Precision, n.Operator.Properly)"), PHRASES,
     "during hour of"),
    ("`properly includes` a point passes UTC",
     (EV, "return evalIntervalProperlyContainsPoint(e.evaluationFrame(), leftIv, right, n.Operator.Precision)",
      "return evalIntervalProperlyContainsPoint(time.UTC, leftIv, right, n.Operator.Precision)"), PHRASES,
     "properly includes hour of"),
    ("`properly included in` a point passes UTC",
     (EV, "return evalIntervalProperlyContainsPoint(e.evaluationFrame(), rightIv, left, n.Operator.Precision)",
      "return evalIntervalProperlyContainsPoint(time.UTC, rightIv, left, n.Operator.Precision)"), PHRASES,
     "properly included in hour of"),
    ("the interval phrases pass UTC",
     (IP, "endpointsOf(e.evaluationFrame(), a, b, op.Precision)", "endpointsOf(time.UTC, a, b, op.Precision)"), PHRASES,
     "want"),
    ("`meets` passes UTC to successorOf: the phrases",
     (IP, "return successorOf(frame, prev, next, precision)", "return successorOf(time.UTC, prev, next, precision)"), PHRASES,
     "meets hour of"),
    ("the window of an unplaced DateTime is read at UTC at its latest end",
     (EV, "high, highOK := temporalCompareRead(frame, hi.left, hi.right, precision, true)",
      "high, highOK := temporalCompareRead(time.UTC, hi.left, hi.right, precision, true)"), WINDOW,
     "before hour of at +05:30"),
    # The harness and the area.
    ("the table gives -03:00 to the new area",
     ("internal/referenceharness/evaloffset/evaloffset.go", '"fractionaloffset": 5*3600 + 30*60,', '"fractionaloffset": -3 * 3600,'), AT_OFFSET,
     "was evaluated at"),
    ("the table gives -03:00 to the new area: the snapshot",
     ("internal/referenceharness/evaloffset/evaloffset.go", '"fractionaloffset": 5*3600 + 30*60,', '"fractionaloffset": -3 * 3600,'), SNAPSHOT,
     "moved in fractionaloffset"),
    ("the snapshot test is at -03:00 for every area",
     ("engine_snapshot_test.go", "time.FixedZone(\"\", evaloffset.Seconds(area))", "time.FixedZone(\"\", evaloffset.Seconds(area)*0-3*3600)"), SNAPSHOT,
     "moved in fractionaloffset"),
    ("the differential test is at -03:00 for every area",
     ("engine_oracle_test.go", "time.FixedZone(\"\", evaloffset.Seconds(area))", "time.FixedZone(\"\", evaloffset.Seconds(area)*0-3*3600)"), REFERENCES,
     "fractionaloffset"),
    ("the comparison tool is at -03:00 for every area",
     ("internal/referenceharness/differential/compare/main.go", "time.FixedZone(\"\", evaloffset.Seconds(area))", "time.FixedZone(\"\", evaloffset.Seconds(area)*0-3*3600)"), COMPARE,
     "timezoneoffset"),
    ("a reference regenerated without the variable",
     ("testdata/snapshots/fractionaloffset.java.json", '"evaluationDateTime" : "2026-01-01T00:00:00+05:30"', '"evaluationDateTime" : "2026-01-01T00:00:00-03:00"'), AT_OFFSET,
     "fractionaloffset.java.json was evaluated at"),
    ("a reference of cql-execution regenerated without the variable",
     ("testdata/snapshots/fractionaloffset.js.json", '"evaluationDateTime": "2026-01-01T00:00:00.000+05:30"', '"evaluationDateTime": "2026-01-01T00:00:00.000-03:00"'), AT_OFFSET,
     "fractionaloffset.js.json was evaluated at"),
    ("references.sh does not tell the harness the offset",
     ("scripts/references.sh", 'offsets="fractionaloffset=+05:30"', 'offsets=""'), SCRIPT,
     "tells the harness"),
    ("the Java harness ignores the variable",
     ("internal/referenceharness/src/Evaluate.java", 'System.getenv("CQL_EVALUATION_OFFSET") != null', "false"), HARNESS,
     "does not read the evaluation offset"),
    ("the node harness ignores the variable",
     ("internal/referenceharness/js/evaluate.js", "process.env.CQL_EVALUATION_OFFSET || '-03:00'", "'-03:00'"), HARNESS,
     "does not read the evaluation offset"),
    # The expected answers.
    ("an expected answer of a test is corrupted",
     ("engine_same_precision_offset_test.go",
      '''{"D6", "@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00",
			[]at{{plus0530, "true"}''', '''{"D6", "@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00",
			[]at{{plus0530, "false"}'''), PROBES,
     "D6 at"),
    ("an answer of the oracle is corrupted",
     ("testdata/fractionaloffset/oracle.tsv", "D41\tBoolean true", "D41\tBoolean false"), ORACLE,
     "D41"),
]

# A deliberate verdict of the area names its row in plan 024 (TestTheDivergencesAreConsistent).
plan = open(os.path.join(repo, "docs/plans/024-same-precision-offset.md")).read()
divergences = [l.split("\t") for l in open(os.path.join(repo, "testdata/snapshots/fractionaloffset.divergences.tsv")).read().split("\n")
               if l and not l.startswith("#")]
named_once = [d[0] for d in divergences if d[2] == "deliberate" and "docs/plans/024-same-precision-offset.md" in d[4]
              and len(re.findall(r"(?<![\w-])" + d[0] + r"(?![\w-])", plan)) == 1 and f" {d[0]}," in plan]
if not named_once:
    print("NO DELIBERATE ROW of plan 024 to mutate, named once in the plan as ` Dn,`")
    sys.exit(2)
mutations.append(
    ("a deliberate row that cites a plan that does not name it",
     ("docs/plans/024-same-precision-offset.md", f" {named_once[0]},", " DX,"), run("TestTheDivergencesAreConsistent"),
     "which does not name " + named_once[0]))

# Code review of plan 024, round 1: the reading at the other value's offset,
# and the offset cleared after each loop of areas.
mutations.append(
    ("the window checks only its two ends",
     ("eval/evaluator.go", "if same, ok := offsetOfTheOtherOf(left, right); ok {",
      "if same, ok := offsetOfTheOtherOf(left, right); ok && false {"),
     run("TestAnUnplacedValueIsUncertainWhereTheWindowDisagrees"),
     "= true; want null"))
mutations.append(
    ("the offset of the last area leaks into what follows",
     ("scripts/references.sh",
      "\t\t# What follows is at the default offset, whatever area came last.\n\t\tunset CQL_EVALUATION_OFFSET\n\t\tjava -cp",
      "\t\tjava -cp"),
     run("TestTheReferencesScriptTellsTheHarnessTheOffsetOfEachArea"),
     "does not clear the offset after a loop of areas"))

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
