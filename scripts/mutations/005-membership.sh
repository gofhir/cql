#!/bin/sh
# Mutations of membership with a null, increment 005
# (docs/plans/005-pertenencia-con-null.md). Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/005-membership.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d)
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()

EVAL = "eval/evaluator.go"
OVERLOAD = "sema/overload.go"
ORACLE = "engine_membership_oracle_test.go"
CORPUS = "testdata/snapshots/membership.cql"


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


# What the engine answers: the cases written by hand and the differential.
ENGINE = run("TestMembershipWithNullFollowsTheSpecification", "TestMembershipWithNullThroughOtherPaths",
             "TestMembershipWithNullInAFHIRList", "TestMembershipWithNullFromTheReferenceELM", "TestAnswersMatchTheReferences")
# What the oracle computes: the verdicts, where all agree, the whole corpus and its controls.
ORACLE_TESTS = run("TestTheMembershipVerdictsFollowTheOracle", "TestTheMembershipOracleAgreesWhereAllAgree",
                   "TestTheMembershipOracleReadsTheWholeCorpus", "TestTheMembershipOracleReadsANestedNullAsAValue")

IN = "\t\treturn fptypes.NewBoolean(listContainsValue(toCollection(right), left)), nil"
CONTAINS = "\treturn fptypes.NewBoolean(listContainsValue(toCollection(left), right)), nil"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The engine.
    ("in answers null for a null element a list does not hold",
     (EVAL, IN, "\t\tif left == nil && !listContainsValue(toCollection(right), nil) {\n\t\t\treturn nil, nil\n\t\t}\n" + IN), ENGINE, "FAIL"),
    ("contains answers null for a null element a list does not hold",
     (EVAL, CONTAINS, "\tif right == nil && !listContainsValue(toCollection(left), nil) {\n\t\treturn nil, nil\n\t}\n" + CONTAINS), ENGINE, "FAIL"),
    ("in finds a null element in every list",
     (EVAL, IN, "\t\treturn fptypes.NewBoolean(left == nil || listContainsValue(toCollection(right), left)), nil"), ENGINE, "FAIL"),
    ("ADR 003, decision 6: two nulls answer null",
     (EVAL, CONTAINS, "\tif left == nil && right == nil {\n\t\treturn nil, nil\n\t}\n" + CONTAINS), ENGINE, "FAIL"),
    ("includes and included in with a null keep the old answer",
     (EVAL, "\tif (left != nil && right != nil) || e.ctx.Plan == nil {",
      "\tif true || (left != nil && right != nil) || e.ctx.Plan == nil {"), ENGINE, "FAIL"),
    # The overload, which the semantic phase decides since plan 006.
    ("the overload read from the value: every list is the membership one",
     (OVERLOAD, "\t\tif c.meets(element, t.Element) {\n\t\t\treturn ElementOverload\n\t\t}",
      "\t\tif true {\n\t\t\treturn ElementOverload\n\t\t}"), ENGINE, "FAIL"),
    ("the list-list overload with a null answers as the membership one",
     (EVAL, "\tif o == sema.ListOverload {", "\tif false && o == sema.ListOverload {"), ENGINE, "FAIL"),
    ("an interval and its point are not the membership overload",
     (OVERLOAD, "\t\tif c.meets(element, t.Point) {", "\t\tif false && c.meets(element, t.Point) {"), ENGINE, "FAIL"),
    ("a precision keeps includes with a null from the membership overload",
     (OVERLOAD, "\tif c.speculative > 0 || op.Properly || op.Offset != \"\" {",
      "\tif c.speculative > 0 || op.Properly || op.Offset != \"\" || op.Precision != \"\" {"), ENGINE, "FAIL"),
    ("properly includes is answered as includes",
     (OVERLOAD, "\tif c.speculative > 0 || op.Properly || op.Offset != \"\" {",
      "\tif c.speculative > 0 || op.Offset != \"\" {"), ENGINE, "FAIL"),
    ("an element that converts implicitly is not the membership overload",
     (OVERLOAD, "\tif ImplicitlyConverts(from, to) {", "\tif Equal(from, to) {"), ENGINE, "FAIL"),
    ("a container that converts to the element is not the membership overload",
     (OVERLOAD, "\treturn c.reaches(a, b) || c.reaches(b, a)", "\treturn c.reaches(a, b)"), ENGINE, "FAIL"),
    ("the list-list overload needs equal list types",
     (OVERLOAD, "\t\tif c.meets(element, t) {", "\t\tif Equal(element, t) {"), ENGINE, "FAIL"),
    ("a list does not convert element by element",
     (OVERLOAD, "\t\treturn ok && c.reaches(f.Element, t.Element)", "\t\treturn ok && Equal(f.Element, t.Element)"),
     ENGINE, "FAIL"),
    ("a FHIR.Period branch of a choice is not read as the interval the model makes it",
     (OVERLOAD, "\t\tif target, isInterval := ParseTypeName(mc.To).(*Interval); isInterval {",
      "\t\tif target, isInterval := ParseTypeName(mc.To).(*Interval); false && isInterval {"), ENGINE, "FAIL"),
    # The oracle.
    ("the oracle: a null inside an element reads as a null element",
     (ORACLE, "\treturn item == \"null\" || memberTypedNull.MatchString(item)", "\treturn strings.Contains(item, \"null\")"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: a null list holds a null",
     (ORACLE, "return membershipVerdict{\"false\", \"a null list holds nothing", "return membershipVerdict{\"true\", \"a null list holds nothing"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: the list-list overload read as the membership one",
     (ORACLE, "return membershipVerdict{\"null\", \"a null list in the list-list overload", "return membershipVerdict{\"false\", \"a null list in the list-list overload"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: ADR 003, decision 6, read the other way",
     (ORACLE, "return membershipVerdict{\"false\", \"a null interval and a null point", "return membershipVerdict{\"null\", \"a null interval and a null point"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: a null point in an interval is false",
     (ORACLE, "return membershipVerdict{\"null\", \"a null point in an interval", "return membershipVerdict{\"false\", \"a null point in an interval"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: a value is never in a list",
     (ORACLE, "return membershipVerdict{answer(slices.Contains(c.items, e.text)),", "return membershipVerdict{answer(false),"),
     ORACLE_TESTS, "FAIL"),
    ("the membership corpus is not the one the references answered",
     (CORPUS, "define D0: ", "define D0: (1 in List<Integer>{ 1 }) or "), run("TestAnswersMatchTheReferences"), "FAIL"),
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
        print(f"SURVIVED     {name}: expected a failure saying {text!r}")
        survived += 1
    else:
        print(f"caught       {name}")
sys.exit(1 if survived else 0)
PY
