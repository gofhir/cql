#!/bin/sh
# Mutations of the oracles of increment 001 (docs/plans/001-decimal-division.md).
# Each is applied to a copy of the repository, and the test it names has to fail.
# Usage: sh scripts/mutations/001-decimal-division.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
work=${1:-$(mktemp -d)}
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()

DIV = "eval/evaluator.go"
ANS = "internal/referenceharness/answers/answers.go"
CONF = "conformance/conformance_test.go"
# (name, file, original text, mutated text, test that has to fail)
mutations = [
    ("an Integer again", DIV,
     "\t\treturn newDecimalFromD(quotient), nil\n\tcase ast.OpMod:",
     "\t\treturn funcs.IntegerOf(quotient), nil\n\tcase ast.OpMod:", "TestConformance"),
    ("Div rounded to 16 places, then truncated", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)", "quotient := ld.Div(rd).Truncate(0)", "TestAnswersMatchTheReferences"),
    ("floored instead of truncated toward zero", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)", "quotient := ld.Div(rd).Floor()", "TestAnswersMatchTheReferences"),
    ("scaled to 8 places (FLOOR) before truncating", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)", "quotient := funcs.ScaleDecimal(ld.DivRound(rd, 20)).Truncate(0)",
     "TestAnswersMatchTheReferences"),
    ("no Decimal range guard", DIV,
     "if !funcs.WithinDecimal(quotient) {", "if false {", "TestATruncatedQuotientPastTheDecimalRangeIsNull"),
    ("the int64 guard kept", DIV,
     "if !funcs.WithinDecimal(quotient) {", "if !funcs.WithinDecimal(quotient) || !quotient.BigInt().IsInt64() {",
     "TestAnswersMatchTheReferences"),
    ("a large quotient moved by one", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)",
     "quotient, _ := ld.QuoRem(rd, 0)\n\t\tif quotient.Abs().GreaterThan(decimal.New(1, 18)) {\n\t\t\tquotient = quotient.Add(decimal.New(1, 0))\n\t\t}",
     "TestAnswersMatchTheReferences"),
    ("a large quotient moved by a hundred thousand", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)",
     "quotient, _ := ld.QuoRem(rd, 0)\n\t\tif quotient.Abs().GreaterThan(decimal.New(1, 18)) {\n\t\t\tquotient = quotient.Add(decimal.New(1, 5))\n\t\t}",
     "TestAnswersMatchTheReferences"),
    ("10^19 answered as 10^19 + 1", DIV,
     "quotient, _ := ld.QuoRem(rd, 0)",
     "quotient, _ := ld.QuoRem(rd, 0)\n\t\tif quotient.Equal(decimal.New(1, 19)) {\n\t\t\tquotient = quotient.Add(decimal.New(1, 0))\n\t\t}",
     "TestAnswersMatchTheSnapshots"),
    ("Same by value only", ANS,
     "\t\tif isDecimal(a) != isDecimal(b) {\n\t\t\treturn false\n\t\t}\n", "", "TestNumbersAreComparedByValueWithinTheirType"),
    ("Same blind to a reference Decimal against an Integer", ANS,
     "\t\tif isDecimal(a) != isDecimal(b) {", "\t\tif !isDecimal(a) && isDecimal(b) {",
     "TestNumbersAreComparedByValueWithinTheirType"),
    ("Same blind inside containers", ANS,
     "\tif isDecimal(a) {\n\t\treturn \"Decimal \" + a.Num.String()\n\t}\n\treturn a.Num.String()",
     "\treturn a.Num.String()", "TestAContainersNumbersKeepTheirType"),
    ("Same tolerant of large whole numbers", ANS,
     "max(writtenDigits(a.Num), writtenDigits(b.Num)) > floatWrittenDigits", "false",
     "TestAFloat64sNumbersAreTheSameAcrossPlatforms"),
    ("SameExactly with the tolerance", ANS,
     "\t\treturn isDecimal(a) == isDecimal(b) && a.Num.Equal(b.Num)\n", "\t\treturn Same(a, b)\n",
     "TestACorpusWithoutFloatsIsComparedExactly"),
    ("valuesEqual by value only", CONF,
     "return got.Equal(want) && sameNumberTypes(got, want)", "return got.Equal(want)",
     "TestValuesEqualTellsAnIntegerFromADecimal"),
    ("an untyped output compared by value", CONF,
     "return func(got, want fptypes.Value) bool { return valuesEqual(got, asDecimals(want)) }",
     "return func(got, want fptypes.Value) bool { return got != nil && want != nil && got.Equal(want) }",
     "TestAnUntypedOutputIsReadAsDecimals"),
]

PACKAGES = [".", "./conformance/", "./internal/referenceharness/answers/", "./eval/", "./funcs/"]


def fresh_copy():
    copy = os.path.join(work, "copy")
    shutil.rmtree(copy, ignore_errors=True)
    for f in files:
        src = os.path.join(repo, f)
        if os.path.isfile(src):
            os.makedirs(os.path.dirname(os.path.join(copy, f)), exist_ok=True)
            shutil.copy2(src, os.path.join(copy, f))
    return copy


def run_tests(copy):
    run = subprocess.run(["go", "test", "-count=1", *PACKAGES], cwd=copy, capture_output=True, text=True)
    out = run.stdout + run.stderr
    failed = {l.split()[2] for l in run.stdout.splitlines() if l.lstrip().startswith("--- FAIL")}
    return run.returncode, ("build failed" in out or "[setup failed]" in out), failed


# A baseline that fails would pass every mutation off as caught.
code, build, _ = run_tests(fresh_copy())
if code != 0:
    print("BASELINE FAILS" + (" TO BUILD" if build else "") + ": the mutations prove nothing")
    sys.exit(2)
print("baseline  passes")

survived = 0
for name, path, old, new, expected in mutations:
    copy = fresh_copy()
    target = os.path.join(copy, path)
    text = open(target).read()
    if text.count(old) != 1:
        print(f"NOT APPLIED  {name}: found {text.count(old)} times")
        survived += 1
        continue
    open(target, "w").write(text.replace(old, new))
    code, build, failed = run_tests(copy)
    if build:
        print(f"BUILD FAILED {name}: a mutation that does not compile proves nothing")
        survived += 1
    elif code == 0 or expected not in failed:
        print(f"SURVIVED     {name}: {expected} did not fail")
        survived += 1
    else:
        print(f"caught       {name}: {', '.join(sorted(f for f in failed if '/' not in f))}")
sys.exit(1 if survived else 0)
PY
