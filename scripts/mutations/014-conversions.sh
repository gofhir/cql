#!/bin/sh
# Mutations of what plan 014 holds (docs/plans/014-todate-toratio.md): ToDate,
# which reads a string with a time as ToDateTime does and keeps the date
# written; ToRatio, which splits at the ':' outside quotes and reads each side
# as ToQuantity; the Ratio literal read the same way; ConvertsToDate and
# ConvertsToRatio, held to their conversion; convert to Date, null where it
# cannot; and the area written by hand that holds them. Each is applied to a
# copy of the repository; the command it names has to fail, and its output has
# to say why.
# Usage: sh scripts/mutations/014-conversions.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-014.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


CONVERSION = run("TestToDateReadsAString", "TestToDateOfADateTime", "TestToDateOfAFHIRDateIsASystemDate",
                 "TestAConversionWithoutAnOverloadIsAnError", "TestToRatioReadsTwoQuantities",
                 "TestARatioLiteralIsARatio", "TestConvertsToFollowsItsConversion",
                 "TestConvertToDateIsNullWhereItCannot")
CONSISTENT = run("TestTheDivergencesAreConsistent")
TYPES = "funcs/type_ops.go"
RATIO = "funcs/ratio_string.go"
EVAL = "eval/evaluator.go"
PLAN = "docs/plans/014-todate-toratio.md"
TODATE_TIME = "\t\tdt, err := cqltypes.NewDateTime(s)\n\t\tif err != nil {\n\t\t\treturn nil, nil\n\t\t}\n\t\treturn dateOf(dt.String())\n"
LITERAL = ("\t\tnumerator, err := ratioSide(left)\n")
SIDE = "\tif err != nil || q.Unit() != \"\" {\n"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # ToDate.
    ("a string with a time read as a Date",
     (TYPES, "\t\tif strings.IndexByte(s, 'T') < 0 {\n", "\t\tif true {\n"), CONVERSION,
     "ToDate('2014-01-01T12:05:05.955') = null, want 2014-01-01"),
    ("the time ignored even when it is no time",
     (TYPES, TODATE_TIME, "\t\treturn dateOf(s)\n"), CONVERSION,
     "ToDate('2014-01-01T25:00:00') = 2014-01-01, want null"),
    ("the date moved to another offset before it is cut",
     (TYPES, "\t\treturn dateOf(dt.String())\n",
      "\t\tt := dt.ToTime().UTC()\n\t\treturn fptypes.NewDate(fmt.Sprintf(\"%04d-%02d-%02d\", t.Year(), int(t.Month()), t.Day()))\n"),
     CONVERSION, "ToDate('2014-01-01T00:30:00+14:00') = 2013-12-31, want 2014-01-01"),
    ("ToDate of a type without an overload answered null",
     (TYPES, '\treturn nil, fmt.Errorf("cannot convert %s to Date", v.Type())\n', "\treturn nil, nil\n"), CONVERSION,
     "ToDate(5) = null, want an error"),
    ("ConvertsToDate that does not know a DateTime",
     (EVAL, "\t\tcase fptypes.Date, fptypes.DateTime, fptypes.String:\n\t\t\tconverted, err := funcs.ToDate(src)\n",
      "\t\tcase fptypes.Date, fptypes.String:\n\t\t\tconverted, err := funcs.ToDate(src)\n"), CONVERSION,
     "ConvertsToDate(@2014-01-01T10:00) = false, want true"),
    ("convert to Date by the cast's conversion",
     (EVAL, "\t\t\tif strings.EqualFold(nt.Name, \"Date\") && (nt.Namespace == \"\" || nt.Namespace == \"System\") {\n\t\t\t\treturn funcs.ToDate(operand)\n\t\t\t}\n", ""),
     CONVERSION, "convert 'abc' to Date = error"),
    # ToRatio.
    ("a Ratio split at every ':'",
     (RATIO, "\t\t\tif quoted {\n\t\t\t\tcontinue\n\t\t\t}\n", ""), CONVERSION, "= null, want 1 '{a:b}'"),
    ("a side that is no Quantity left unread",
     (RATIO, "\tif err != nil || denominator == nil {\n", "\tif err != nil {\n"), CONVERSION,
     "ToRatio('1:') = "),
    ("ToRatio of a Ratio answered null",
     (RATIO, "\tcase cqltypes.Ratio:\n\t\treturn val, nil\n", "\tcase cqltypes.Ratio:\n\t\treturn nil, nil\n"), CONVERSION,
     "ConvertsToRatio(1 'mg':2 'mg') = false, want true"),
    ("ConvertsToRatio that does not follow ToRatio",
     (EVAL, "funcs.ToRatio(src)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\treturn fptypes.NewBoolean(converted != nil), nil\n",
      "funcs.ToRatio(src)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\treturn fptypes.NewBoolean(converted != nil || strings.Contains(src.String(), \":\")), nil\n"), CONVERSION,
     "but ToRatio("),
    ("the Ratio literal left a String",
     (EVAL, "\tcase ast.LiteralRatio:\n", "\tcase ast.LiteralRatio + 100:\n"), CONVERSION,
     "(1 'mg':2 'mg') is Ratio = false, want true"),
    ("the Ratio literal read by ToRatio's string grammar",
     (EVAL, LITERAL, "\t\tnumerator, err := funcs.QuantityFromString(left)\n\t\t_ = ratioSide\n"), CONVERSION,
     "(1 day:2 days).numerator = "),
    ("a side of a Ratio literal without the default unit",
     (EVAL, SIDE, "\tif err != nil || true {\n"), CONVERSION, "(1:2).denominator = 2, want 2 '1'"),
    ("ToString of a Ratio by its own text",
     (TYPES, "\tif r, ok := v.(cqltypes.Ratio); ok && r.Numerator != nil && r.Denominator != nil {\n",
      "\tif r, ok := v.(cqltypes.Ratio); ok && false && r.Numerator != nil {\n"), CONVERSION,
     "ToRatio(ToString(1 'day':2 'days')) = (1 'day':2 'days') = null, want true"),
    ("convert to String of a Ratio by its own text",
     (EVAL, "\t\t\t\treturn funcs.ToString(operand), nil\n", "\t\t\t\t_ = 0\n"), CONVERSION,
     "convert (1 'day':2 'days') to String = 1 day : 2 days"),
    ("convert to Ratio by the cast's conversion",
     (EVAL, "\t\t\tif strings.EqualFold(nt.Name, \"Ratio\") && (nt.Namespace == \"\" || nt.Namespace == \"System\") {\n\t\t\t\treturn funcs.ToRatio(operand)\n\t\t\t}\n", ""),
     CONVERSION, "convert 'abc' to Ratio = error"),
    ("ToDate of a FHIR.date that keeps its FHIR type",
     (TYPES, "\tcase fptypes.Date, fptypes.DateTime:\n\t\treturn dateOf(val.String())\n",
      "\tcase fptypes.Date:\n\t\treturn val, nil\n\tcase fptypes.DateTime:\n\t\treturn dateOf(val.String())\n"), CONVERSION,
     "FHIR primitive true"),
    # The area and its classification.
    ("a deliberate divergence the plan does not name",
     (PLAN, "D166, D167, D168 y D249.", "D166, D167 y D168."), CONSISTENT, "does not name D249"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"conversions", 265, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "conversions.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", "limits limitsexp conversions", "limits limitsexp"),
     run("TestEveryAreaIsRunByTheReferences"), "scripts/references.sh does not run it"),
    ("a test of G3f back in the suite's area",
     ("testdata/snapshots/suitegaps.cql", "\ndefine D46:", "\ndefine D49: ToDate('2014-01-01') // ToDate1\ndefine D46:"),
     run("TestEveryKnownFailureIsOfTheSuite", package="./conformance/"),
     "holds ToDate1, which is not in knownFailures"),
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
