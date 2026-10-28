#!/bin/sh
# Mutations of what plan 011 holds for the Decimal literal
# (docs/plans/011-literal-decimal.md): the error past what cql-tests reads,
# its two bounds counted in digits of value, the Quantity literal left out of
# the check, and the area written by hand that holds them. Each is applied to
# a copy of the repository; the command it names has to fail, and its output
# has to say why.
# Usage: sh scripts/mutations/011-literal.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-011.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


SNAPSHOTS = run("TestAnswersMatchTheSnapshots")
CONSISTENT = run("TestTheDivergencesAreConsistent")
CONFORMANCE = run("TestConformance", package="./conformance/")
BOUNDS = run("TestADecimalLiteralIsBoundedByItsDigitsOfValue", package="./eval/")
LIBRARY = run("TestADecimalLiteralPastTheSuiteFailsWhereItIsEvaluated")
EVAL = "eval/evaluator.go"
PLAN = "docs/plans/011-literal-decimal.md"
MOVED = "answers moved in decimalliterals"
CHECK = "\t\tif err := validateCQLDecimal(n.Value); err != nil {\n\t\t\treturn nil, err\n\t\t}\n"


# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("the error not raised",
     (EVAL, CHECK, ""), SNAPSHOTS, MOVED),
    ("the error not raised, against the suite",
     (EVAL, CHECK, ""), CONFORMANCE, "DecimalTenthStep"),
    ("the error not raised, for the library",
     (EVAL, CHECK, ""), LIBRARY, "want the literal's error"),
    ("rounded to the scale instead of an error",
     (EVAL, CHECK, "\t\tif err := validateCQLDecimal(n.Value); err != nil {\n"
      "\t\t\td, _ := decimal.NewFromString(n.Value)\n\t\t\treturn newDecimalFromD(funcs.ScaleDecimal(d)), nil\n\t\t}\n"),
     SNAPSHOTS, MOVED),
    ("nine places of value allowed",
     (EVAL, 'if len(strings.TrimRight(places, "0")) > 8 {', 'if len(strings.TrimRight(places, "0")) > 9 {'), SNAPSHOTS, MOVED),
    ("29 integer digits allowed",
     (EVAL, 'if len(strings.TrimLeft(whole, "0")) > 28 {', 'if len(strings.TrimLeft(whole, "0")) > 29 {'), SNAPSHOTS, MOVED),
    ("the trailing zeros counted (decision D)",
     (EVAL, 'len(strings.TrimRight(places, "0"))', "len(places)"), SNAPSHOTS, MOVED),
    ("the leading zeros counted (decision D)",
     (EVAL, 'len(strings.TrimLeft(whole, "0"))', "len(whole)"), SNAPSHOTS, MOVED),
    ("the trailing zeros counted, against the hand cases",
     (EVAL, 'len(strings.TrimRight(places, "0"))', "len(places)"), BOUNDS, "want ok true"),
    ("the sign not read",
     (EVAL, 'strings.Cut(strings.TrimLeft(s, "+-"), ".")', 'strings.Cut(s, ".")'), BOUNDS, "want ok true"),
    ("the check applied to the Quantity literal",
     (EVAL, "\tcase ast.LiteralQuantity:\n\t\treturn fptypes.NewQuantity(n.Value)",
      "\tcase ast.LiteralQuantity:\n\t\tif err := validateCQLDecimal(strings.Fields(n.Value)[0]); err != nil {\n"
      "\t\t\treturn nil, err\n\t\t}\n\t\treturn fptypes.NewQuantity(n.Value)"),
     CONFORMANCE, "QuantityFractionalTooBig"),
    # The area and its classification.
    ("a deliberate divergence the plan does not name",
     (PLAN, "D44 y D46.", "D44."), CONSISTENT, "does not name D46"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"decimalliterals", 60, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "decimalliterals.cql is not in snapshotAreas"),
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
