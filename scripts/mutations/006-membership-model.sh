#!/bin/sh
# Mutations of the overload of includes, included in and during through the
# model, increment 006 (docs/plans/006-sobrecarga-con-el-modelo.md). Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/006-membership-model.sh [work-directory]
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
CHECK = "sema/check.go"
OVERLOAD = "sema/overload.go"
ORACLE = "engine_fhirmembership_oracle_test.go"
CASES = "engine_membership_model_test.go"
NODES = "testdata/snapshots/fhirmembership.nodes.tsv"


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


# What the engine answers: the cases written by hand, plan 005's, and the differential.
ENGINE = run("TestMembershipThroughTheModelFollowsTheSpecification", "TestMembershipWithNullFollowsTheSpecification",
             "TestMembershipWithNullInAFHIRList", "TestAnswersMatchTheReferences")
# What the oracle computes: the verdicts, where all agree, and the whole corpus.
ORACLE_TESTS = run("TestTheFHIRMembershipVerdictsFollowTheOracle", "TestTheFHIRMembershipOracleAgreesWhereAllAgree",
                   "TestTheFHIRMembershipOracleReadsTheWholeCorpus")

RECORD = "\t\tc.overloads[e] = decided"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The semantic phase.
    ("the semantic phase records no overload",
     (OVERLOAD, RECORD, "\t\t_ = decided"), ENGINE, "FAIL"),
    ("every overload recorded is the element one",
     (OVERLOAD, RECORD, "\t\tc.overloads[e] = ElementOverload"), ENGINE, "FAIL"),
    ("a value reaches a list of it, and a list of one its element",
     (OVERLOAD, "\tif ImplicitlyConverts(from, to) {",
      "\tif l, ok := to.(*List); ok && c.reaches(from, l.Element) {\n\t\treturn true\n\t}\n"
      "\tif l, ok := from.(*List); ok && c.reaches(l.Element, to) {\n\t\treturn true\n\t}\n"
      "\tif ImplicitlyConverts(from, to) {"), ENGINE, "FAIL"),
    ("a model conversion is chained with a System one",
     (OVERLOAD, "\t\t\tif Equal(ParseTypeName(mc.To), to) {",
      "\t\t\tif Equal(ParseTypeName(mc.To), to) || ImplicitlyConverts(ParseTypeName(mc.To), to) {"), ENGINE, "FAIL"),
    ("no model conversion reaches",
     (OVERLOAD, "\t\t\tif Equal(ParseTypeName(mc.To), to) {", "\t\t\tif false && Equal(ParseTypeName(mc.To), to) {"), ENGINE, "FAIL"),
    ("Any meets every type",
     (OVERLOAD, "\tif ImplicitlyConverts(from, to) {", "\tif ImplicitlyConverts(from, to) || Equal(to, Any) {"),
     run("TestMembershipWithAnAnyOperandStaysUndecided"), "FAIL"),
    ("the first branch of a choice decides, not the element one",
     (OVERLOAD, "o != 0 && (decided == 0 || o == ElementOverload)", "o != 0 && decided == 0"), ENGINE, "FAIL"),
    ("a choice is not reached by its branches",
     (OVERLOAD, "\tif choice, ok := to.(*Choice); ok {", "\tif choice, ok := to.(*Choice); false && ok {"), ENGINE, "FAIL"),
    ("a choice reaches by none of its branches",
     (OVERLOAD, "\t\t\tif c.reaches(branch, to) {", "\t\t\tif false && c.reaches(branch, to) {"), ENGINE, "FAIL"),
    ("an interval does not convert point by point",
     (OVERLOAD, "\t\treturn ok && c.reaches(f.Point, t.Point)", "\t\treturn ok && Equal(f.Point, t.Point)"), ENGINE, "FAIL"),
    ("a choice container is read whole, not by its branches",
     (OVERLOAD, "\tfor _, ct := range branches(container) {", "\tfor _, ct := range []Type{container} {"), ENGINE, "FAIL"),
    ("a definition reached first by a speculative pass keeps none of its findings",
     (CHECK, "\tc.speculative = 0\n", ""), run("TestMembershipInADefinitionReachedFirstFromACallIsDecided"), "FAIL"),
    # The evaluator.
    ("the evaluator ignores the recorded overload",
     (EVAL, "\to, ok := e.ctx.Plan.OverloadOf(n)", "\to, ok := sema.Overload(0), false"), ENGINE, "FAIL"),
    # The cases written by hand: the node the translator wrote.
    ("a case says the list-list overload is the element one",
     (CASES, "{\"Patient.name.given includes (null as List<String>)\", nodeIncludes,",
      "{\"Patient.name.given includes (null as List<String>)\", nodeContains,"),
     run("TestMembershipThroughTheModelFollowsTheSpecification"), "FAIL"),
    # The oracle.
    ("the oracle: the list-list node is read as the membership one",
     (ORACLE, "\t\tnodes[name] = node", "\t\tnodes[name] = strings.NewReplacer(\"IncludedIn\", \"In\", \"Includes\", \"Contains\").Replace(node)"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: a list holds no null",
     (ORACLE, "\tif slices.Contains(c.items, \"null\") {", "\tif false {"), ORACLE_TESTS, "FAIL"),
    ("the oracle: a Period present is no interval",
     (ORACLE, "\t\t\treturn field, suffix == \"Period\"", "\t\t\treturn field, false"), ORACLE_TESTS, "FAIL"),
    ("ADR 003, decision 6, read the other way by the oracle",
     (ORACLE, "return membershipVerdict{\"false\", \"a null container and a null element", "return membershipVerdict{\"null\", \"a null container and a null element"),
     ORACLE_TESTS, "FAIL"),
    ("the oracle: a refused expression is answered",
     (ORACLE, "return membershipVerdict{\"refused\",", "return membershipVerdict{\"null\","), ORACLE_TESTS, "FAIL"),
    ("the nodes are not the translator's",
     (NODES, "D2\trejected", "D2\tIn"), ORACLE_TESTS, "FAIL"),
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
