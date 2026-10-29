#!/bin/sh
# Mutations of what plan 013 holds (docs/plans/013-null-donde-va-null.md):
# null where a step, Exp, Ln or a division by zero cannot be represented, the
# one error every type gives for a step past its range, the start and end of
# an interval open at the edge, meets at the last value of a type, and the
# area written by hand that holds them. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/013-null.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-013.XXXXXX")
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
EDGE = run("TestAStepPastItsTypeIsNull", "TestTheStartOfABoundaryAtTheEdgeIsNull",
           "TestTheStartOfANamedIntervalAtTheEdgeIsNull", "TestAStepThatIsNoStepStaysAnError",
           "TestMeetsAtTheEdgeOfAType", "TestWhatCannotBeRepresentedIsNull")
STEP = run("TestAStepOffTheTypeIsErrPastRange", package="./types/")
RUNTIME = run("TestRuntimeMatchesTheReference")
RANGE = "eval/integer_range.go"
EVAL = "eval/evaluator.go"
SUCC = "types/successor.go"
MATH = "funcs/math_ops.go"
PLAN = "docs/plans/013-null-donde-va-null.md"
NULLED = "\tif errors.Is(err, cqltypes.ErrPastRange) {\n\t\treturn nil, nil\n\t}\n"
DECIMAL = "\tcase fptypes.Decimal:\n\t\tif !funcs.WithinDecimal(x.Value()) {\n\t\t\treturn nil, nil\n\t\t}\n"
QUANTITY = "\tcase fptypes.Quantity:\n\t\tif !funcs.WithinDecimal(x.Value()) {\n\t\t\treturn nil, nil\n\t\t}\n"
START = "\t\t\tv, err := iv.Start()\n\t\t\treturn e.stepWithinItsType(n, v, err)\n"
EXP_RANGE = "\tif !WithinDecimal(exp) {\n\t\treturn nil, nil\n\t}\n"
LN = "\tif d.Sign() <= 0 {\n\t\treturn nil, nil\n\t}\n"
QDIV = "\t\t\tif rd.IsZero() {\n\t\t\t\treturn nil, nil\n\t\t\t}\n\t\t\treturn lq.Divide(rd)\n"


# name, (path, old, new), command, text the failure has to hold
mutations = [
    # Successor and predecessor (decision A).
    ("ErrPastRange left an error",
     (RANGE, NULLED, NULLED.replace("return nil, nil", "return nil, err")), CONFORMANCE, "SuccessorOverflowT"),
    ("a Decimal step that stays near the least, within a float's tolerance",
     (SUCC, "\t\treturn newDecimal(t.Value().Sub(decimalStep))\n",
      "\t\tif t.Value().LessThan(decimal.NewFromInt(-1000000000000000)) {\n\t\t\treturn t, nil\n\t\t}\n"
      "\t\treturn newDecimal(t.Value().Sub(decimalStep))\n"),
     SNAPSHOTS, "answers moved in limits"),
    ("ErrPastRange left an error, over the area",
     (RANGE, NULLED, NULLED.replace("return nil, nil", "return nil, err")), SNAPSHOTS, "answers moved in limits"),
    ("a Decimal past the largest kept",
     (RANGE, DECIMAL, ""), EDGE, "successor of maximum Decimal = 100000000000000000000, want null"),
    ("a Quantity past the largest Decimal kept",
     (RANGE, QUANTITY, ""), EDGE, "'mg', want null"),
    ("the 32-bit edge not held",
     (RANGE, "\treturn e.withinInteger(node, v, nil)\n}", "\treturn v, nil\n}"), EDGE,
     "successor of 2147483647 = 2147483648, want null"),
    ("the calendar's refusal not read as the edge",
     (SUCC, "if errors.Is(err, fptypes.ErrInvalidTemporal) {", "if false {"), STEP, "want ErrPastRange"),
    ("the calendar's refusal not read as the edge, against the suite",
     (SUCC, "if errors.Is(err, fptypes.ErrInvalidTemporal) {", "if false {"), CONFORMANCE, "SuccessorOverflowDt"),
    ("the clock's edge an error of its own",
     (SUCC, "\t\t\treturn nil, stepError(t, delta)\n", "\t\t\treturn nil, fmt.Errorf(\"past the clock\")\n"),
     CONFORMANCE, "PredecessorUnderflowT"),
    # Start and end (decision C).
    ("start of not stepped within the type",
     (EVAL, START, "\t\t\treturn iv.Start()\n"), EDGE, "start of Interval(2147483647, null]"),
    ("start of an Integer not certain",
     (RANGE, "\tcase ast.OpStartOf, ast.OpEndOf:\n", "\tcase ast.OpWidthOf:\n"), EDGE,
     "start of Interval(2147483647, null] = 2147483648"),
    ("the start of a named interval not certain",
     (RANGE, "\tcase *ast.IdentifierRef:\n\t\tstmt := e.definitionOf(x)\n\t\tif stmt == nil {\n\t\t\treturn false\n\t\t}\n\t\tkey :=",
      "\tcase *ast.IdentifierRef:\n\t\tstmt := e.definitionOf(x)\n\t\tif stmt == nil || true {\n\t\t\treturn false\n\t\t}\n\t\tkey :="),
     EDGE, "start of I, with"),
    ("a boundary settled under a definition's name",
     (RANGE, "\t\t\tkey = settled{settledLow, x.Name}\n", "\t\t\tkey = settled{settledInteger, \"low of \" + x.Name}\n"),
     EDGE, 'successor of "low of I"'),
    ("an error that is no edge read as null",
     (RANGE, NULLED + "\tif err != nil {\n\t\treturn nil, err\n\t}\n", NULLED + "\tif err != nil {\n\t\treturn nil, nil\n\t}\n"),
     EDGE, "evaluated; want an error"),
    ("a type without a successor read as one at the edge",
     ("funcs/interval.go", "\t\tif cqltypes.HasSuccessor(aHigh) {\n\t\t\treturn false\n\t\t}\n", "\t\treturn false\n"), EDGE,
     "meets before Interval['a', 'a'] = false, want true"),
    ("meets read by equality at the edge",
     ("funcs/interval.go", "\t\tif cqltypes.HasSuccessor(aHigh) {\n\t\t\treturn false\n\t\t}\n", ""), EDGE,
     "meets before Interval[@T23:59:59.999, @T23:59:59.999] = true, want false"),
    # Exp, Ln and the division (decision B).
    ("Exp past the largest Decimal kept",
     (MATH, EXP_RANGE, ""), EDGE, "Exp(100) = "),
    ("Exp past the largest Decimal kept, over the arithmetic area",
     (MATH, EXP_RANGE, ""), SNAPSHOTS, "answers moved in arithmetic"),
    ("Exp past a float an error",
     (MATH, "\tif math.IsInf(result, 0) || math.IsNaN(result) {\n\t\treturn nil, nil\n\t}\n",
      "\tif math.IsInf(result, 0) || math.IsNaN(result) {\n\t\treturn nil, fmt.Errorf(\"overflow\")\n\t}\n"),
     CONFORMANCE, "Exp1000"),
    ("Ln of zero an error",
     (MATH, LN, "\tif d.IsZero() {\n\t\treturn nil, fmt.Errorf(\"zero\")\n\t}\n\tif d.Sign() < 0 {\n\t\treturn nil, nil\n\t}\n"),
     CONFORMANCE, "Ln0"),
    ("a Quantity divided by zero an error",
     (EVAL, QDIV, "\t\t\treturn lq.Divide(rd)\n"), CONFORMANCE, "DivideQuantityByZero"),
    # The area, its classification and the references.
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"limits", 96, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "limits.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", "suitegaps limits limitsexp", "suitegaps limitsexp"),
     run("TestEveryAreaIsRunByTheReferences"), "limits"),
    ("a deliberate divergence the plan does not name",
     (PLAN, "- D106, `Ln(0.00000001)`.", "- `Ln(0.00000001)`."), CONSISTENT, "does not name D106"),
    ("the Java engine's TypeOverflow taken for the oracle",
     ("engine_runtime_reference_test.go",
      '\t\t\t"IntegerSuccessorOfMaximum": "it raises TypeOverflow where the specification answers null",\n', ""),
     RUNTIME, "IntegerSuccessorOfMaximum"),
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
