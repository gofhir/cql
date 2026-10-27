#!/bin/sh
# Mutations of what plan 009 puts in to move to shopspring/decimal v1.6.0
# (docs/plans/009-decimal-1-6.md): the grammar a string is read by as a
# Decimal, the magnitude read from its digits with bounded resources, the null
# of a string that is no Decimal, the cast that still fails on one, and the
# area written by hand that holds them. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/009-decimal.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-009.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


SNAPSHOTS = run("TestAnswersMatchTheSnapshots")
REFERENCES = run("TestAnswersMatchTheReferences")
CONSISTENT = run("TestTheDivergencesAreConsistent")
BUDGET = ["go", "test", "-count=1", "-run", "^TestAStringOfMillionsOfDigitsIsReadInBoundedResources$", "./funcs/"]
READ = "funcs/decimal_scale.go"
DS = "testdata/snapshots/decimalstrings.divergences.tsv"
PLAN = "docs/plans/009-decimal-1-6.md"
MOVED = "answers moved in decimalstrings"


# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The grammar.
    ("a Unicode minus read as a sign",
     (READ, "splitDecimal(s)\n", "splitDecimal(strings.Replace(s, \"\\u2212\", \"-\", 1))\n"), SNAPSHOTS, MOVED),
    ("a fullwidth digit read as its ASCII one",
     (READ, "splitDecimal(s)\n", "splitDecimal(strings.ReplaceAll(s, \"\\uff11\", \"1\"))\n"), SNAPSHOTS, MOVED),
    ("a sign after the point",
     (READ, "\t\tk := digitsFrom(s, j+1)\n\t\tfraction = s[j+1 : k]",
      "\t\tk := j + 1\n\t\tif k < len(s) && (s[k] == '+' || s[k] == '-') {\n\t\t\tk++\n\t\t}\n\t\tk = digitsFrom(s, k)\n\t\tfraction = s[j+1 : k]"),
     SNAPSHOTS, MOVED),
    ("an exponent with no digits",
     (READ, "\t\tif end == k {\n", "\t\tif false && end == k {\n"), SNAPSHOTS, MOVED),
    ("something left after the exponent",
     (READ, "exponent, j == len(s)", "exponent, true"), SNAPSHOTS, MOVED),
    ("the exponent read in 64 bits",
     (READ, "strconv.ParseInt(sign+\"0\"+text, 10, 32)", "strconv.ParseInt(sign+\"0\"+text, 10, 64)"), SNAPSHOTS, MOVED),
    ("the exponent read whole, leading zeros and all",
     (READ, "\ttext = strings.TrimLeft(text, \"0\")\n\tif len(text) > 10 {\n\t\treturn 0, false\n\t}\n", ""), BUDGET,
     "--- FAIL: TestAStringOfMillionsOfDigitsIsReadInBoundedResources"),
    # The magnitude.
    ("the value read by the decimal library first",
     (READ, "\tnegative, whole, fraction, exponentText, ok := ",
      "\tif _, err := decimal.NewFromString(s); err != nil {\n\t\treturn nil, err\n\t}\n\tnegative, whole, fraction, exponentText, ok := "),
     SNAPSHOTS, MOVED),
    ("the last digit kept at the opposite power",
     (READ, "decimal.NewFromBigInt(coefficient, int32(power))", "decimal.NewFromBigInt(coefficient, int32(-power))"),
     SNAPSHOTS, MOVED),
    ("the rest past the scale ignored",
     (READ, "rest = nonZeroFrom(", "rest = false && nonZeroFrom("), SNAPSHOTS, MOVED),
    ("the rest told by the last digit alone",
     (READ, "rest = nonZeroFrom(whole, fraction, max(first, last+1))",
      "rest = digit(count-1) != '0' && nonZeroFrom(whole, fraction, max(first, last+1))"), SNAPSHOTS, MOVED),
    ("a negative floored by no step",
     (READ, "d = d.Sub(decimal.New(1, -DecimalScale))", "d = d.Sub(decimal.New(0, -DecimalScale))"), SNAPSHOTS, MOVED),
    ("the largest Decimal with a rest past it",
     (READ, "if d.GreaterThan(maxDecimal) || (d.Equal(maxDecimal) && rest) {", "if d.GreaterThan(maxDecimal) {"),
     SNAPSHOTS, MOVED),
    ("a zero written with a place",
     (READ, 'return fptypes.NewDecimal("0")', 'return fptypes.NewDecimal("0.0")'), SNAPSHOTS, MOVED),
    ("every digit of the mantissa kept",
     (READ, "\tif written < -DecimalScale {", "\tif false && written < -DecimalScale {"), BUDGET,
     "--- FAIL: TestAStringOfMillionsOfDigitsIsReadInBoundedResources"),
    # A string that is no Decimal.
    ("convert fails on a string that is no Decimal",
     ("eval/evaluator.go", "\t\t\tif errors.Is(err, funcs.ErrNotADecimal) {\n\t\t\t\treturn nil, nil // as ToDecimal",
      "\t\t\tif false && errors.Is(err, funcs.ErrNotADecimal) {\n\t\t\t\treturn nil, nil // as ToDecimal"),
     SNAPSHOTS, MOVED),
    ("ToDecimal fails on a string that is no Decimal",
     ("eval/evaluator.go", "\t\t\tif errors.Is(err, funcs.ErrNotADecimal) {\n\t\t\t\treturn nil, nil // see",
      "\t\t\tif false && errors.Is(err, funcs.ErrNotADecimal) {\n\t\t\t\treturn nil, nil // see"),
     SNAPSHOTS, MOVED),
    ("cast of a string that is no Decimal answers null",
     (READ, "splitDecimal(s)\n\tif !ok {\n\t\treturn nil, fmt.Errorf(\"%w: %.40s\", ErrNotADecimal, s)\n\t}",
      "splitDecimal(s)\n\tif !ok {\n\t\treturn nil, nil\n\t}"),
     SNAPSHOTS, "answers moved in types"),
    # The area.
    ("a deliberate divergence dropped from its file",
     (DS, "\nD51\tjava\tdeliberate\t", "\nD51-gone\tjava\tdeliberate\t"), REFERENCES, "are not the ones recorded"),
    ("a deliberate divergence the plan does not name",
     (PLAN, "D51, D53, D54, D56, D57 y D59 (Java)", "D51, D54, D56, D57 y D59 (Java)"), CONSISTENT, "does not name D53"),
    ("an own defect expected as this engine answers",
     (DS, "\nD12\tjava\town defect\tnull\t", "\nD12\tjava\town defect\tDecimal 0.5\t"),
     CONSISTENT, "an own defect expects an answer other than this engine's"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"decimalstrings", 216, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "decimalstrings.cql is not in snapshotAreas"),
    ("the area with another number of definitions",
     ("engine_snapshot_test.go", '{"decimalstrings", 216, true},', '{"decimalstrings", 215, true},'), SNAPSHOTS,
     "the manifest says 215"),
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
