#!/bin/sh
# Mutations of the evidence schema of the divergence manifests, increment 003
# PR 1 (docs/plans/003-clasificar-divergencias.md). Each is applied to a copy of
# the repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/003-evidence.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    # A directory of its own, removed when the script ends: each run copied the
    # repository into one and left it, until the disk was full.
    work=$(mktemp -d)
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()

ORACLE = "engine_oracle_test.go"
INTEGER = "testdata/snapshots/integer.divergences.tsv"
DIVISION = "testdata/snapshots/division.divergences.tsv"
REFS = ["go", "test", "-count=1", "-run", "TestAnswersMatchTheReferences", "."]
CONSISTENT = ["go", "test", "-count=1", "-run", "TestTheDivergencesAreConsistent", "."]
APPEND = "append"
WRITE = "write"
DELETE = "delete"


def set_field(path, name, reference, column, value):
    # Rewrites one column of one row of a manifest.
    def edit(copy):
        target = os.path.join(copy, path)
        lines = open(target).read().split("\n")
        for i, line in enumerate(lines):
            f = line.split("\t")
            if len(f) == 7 and f[0] == name and f[1] == reference:
                f[column] = value(f)
                lines[i] = "\t".join(f)
                break
        else:
            raise SystemExit(f"{path}: no row {name} {reference}")
        open(target, "w").write("\n".join(lines))
    return edit


def edits(*steps):
    def run(copy):
        for step in steps:
            step(copy)
    return run


def replace(path, old, new):
    def edit(copy):
        target = os.path.join(copy, path)
        text = open(target).read()
        if text.count(old) != 1:
            raise SystemExit(f"{path}: {old!r} found {text.count(old)} times")
        open(target, "w").write(text.replace(old, new))
    return edit


# (name, edits, command, text its output has to hold)
mutations = [
    ("the engine's answer recorded for a divergence changed, the divergence still there",
     set_field(INTEGER, "D0", "java", 6, lambda f: f[6] + " changed"), REFS, "changed: D0"),
    ("a verdict kept when its answers moved",
     replace(ORACLE, "if !ok || !samePins(d.answerPair, found[key], exact) {",
             "if !ok {"),
     ["go", "test", "-count=1", "-run", "TestAChangedAnswerLosesItsVerdict", "."], "its answers moved"),
    ("answers pinned cut, as messages cut them",
     replace(ORACLE, "Engine: pinned(answer.Text)}", "Engine: oneLine(answer.Text)}"), REFS, "changed:"),
    ("a reference defect expecting another answer than this engine's",
     set_field(INTEGER, "D0", "java", 3, lambda f: "Integer 424242"), CONSISTENT, "a reference defect expects"),
    ("an own defect expecting this engine's answer",
     set_field(INTEGER, "D0", "java", 2, lambda f: "own defect"), CONSISTENT, "an own defect expects"),
    ("a deliberate difference citing no decision",
     set_field(DIVISION, "D11", "java", 4, lambda f: f[4].replace("(docs/plans/001-decimal-division.md)", "")),
     CONSISTENT, "cites the ADR or plan"),
    ("a class that is none",
     set_field(INTEGER, "D0", "java", 2, lambda f: "probably fine"), CONSISTENT, "is no class"),
    ("a verdict without evidence",
     set_field(INTEGER, "D0", "java", 4, lambda f: ""), CONSISTENT, "without evidence"),
    ("an undecided divergence without its two readings",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4, lambda f: "unknown"),
     CONSISTENT, "records its two readings"),
    ("a deliberate difference citing a plan that does not exist",
     set_field(DIVISION, "D11", "java", 4, lambda f: f[4].replace("docs/plans/001-decimal-division.md", "docs/plans/999-nonexistent.md")),
     CONSISTENT, "which is not in the repository"),
    ("a deliberate difference citing a plan by a prefix",
     set_field(DIVISION, "D11", "java", 4, lambda f: f[4].replace("docs/plans/001-decimal-division.md", "docs/plans/001")),
     CONSISTENT, "by its whole name"),
    ("a deliberate difference expecting another answer than the one decided",
     set_field(DIVISION, "D11", "java", 3, lambda f: "Decimal 1"), CONSISTENT, "expects this engine's answer"),
    ("answers pinned with their spaces collapsed",
     replace(ORACLE, "return strings.NewReplacer(`\\`, `\\\\`, \"\\t\", `\\t`, \"\\n\", `\\n`, \"\\r\", `\\r`).Replace(s)",
             "return strings.Join(strings.Fields(s), \" \")"), REFS, "changed:"),
    ("pins compared exactly in every area",
     replace(ORACLE, "\tif exact {\n\t\treturn false\n\t}\n\tda, okA", "\tif exact || true {\n\t\treturn false\n\t}\n\tda, okA"),
     ["go", "test", "-count=1", "-run", "TestAPinIsTheSameAcrossPlatforms", "."], "samePin"),
    ("a reading whose source is empty",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/scope.answers.tsv]", "[source: ]")),
     CONSISTENT, "records its two readings"),
    ("a reading whose source does not exist",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/scope.answers.tsv]", "[source: testdata/snapshots/nothing.tsv]")),
     CONSISTENT, "which is not in the repository"),
    ("readings delimited by parentheses inside the answers",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4,
               lambda f: 'reading 1: "x(y)" (); reading 2: "a(b)" ()'),
     CONSISTENT, "records its two readings"),
    ("a deliberate difference citing a URL that holds a plan's name",
     set_field(DIVISION, "D11", "java", 4,
               lambda f: f[4].replace("(docs/plans/001-decimal-division.md)", "(https://example.invalid/docs/plans/001-decimal-division.md)")),
     CONSISTENT, "by its whole name"),
    ("a deliberate difference citing a plan that does not decide it",
     set_field(DIVISION, "D11", "java", 4,
               lambda f: f[4].replace("docs/plans/001-decimal-division.md", "docs/plans/002-adr-semantica-cql.md")),
     CONSISTENT, "which does not name D11"),
    ("a reading whose source is a file outside the corpora",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/scope.answers.tsv]", "[source: go.mod]")),
     CONSISTENT, "which is not a file of testdata/snapshots"),
    ("a deliberate difference citing a plan that names it only inside another name",
     edits(set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 2, lambda f: "deliberate"),
           set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 3, lambda f: f[6]),
           set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 4, lambda f: "see (docs/plans/001-decimal-division.md)")),
     CONSISTENT, "which does not name D1"),
    ("the types oracle ignoring the element of a generic type",
     replace("engine_types_oracle_test.go",
             "return a.name == b.name && len(a.element) == 1 && len(b.element) == 1 && derivesFrom(a.element[0], b.element[0])",
             "return a.name == b.name"),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle taking a Long for an Integer",
     replace("engine_types_oracle_test.go", '"5L":                        {runtime: ptr(named("Long")), answer: "Long 5"},',
             '"5L":                        {runtime: ptr(named("Integer")), answer: "Integer 5"},'),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle deciding null's run-time type",
     replace("engine_types_oracle_test.go",
             'return typesVerdict{undecided: true, reason: "the specification does not say what null\'s run-time type is"}, nil',
             'return typesVerdict{expected: "false", reason: "null is of nothing"}, nil'),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle swapping the classes",
     replace("engine_types_oracle_test.go", "\td.Class = ownDefect\n\tif expectedMatches(d.Expected, d.Engine) {\n\t\td.Class = referenceDefect",
             "\td.Class = referenceDefect\n\tif expectedMatches(d.Expected, d.Engine) {\n\t\td.Class = ownDefect"),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle missing a literal of the corpus",
     replace("engine_types_oracle_test.go", '"5 \'mg\'":                    {runtime: ptr(named("Quantity")), answer: "5 \'mg\'"},', ""),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleReadsTheWholeCorpus", "."], "a literal the oracle does not know"),
    ("a types verdict that does not cite the oracle",
     edits(set_field("testdata/snapshots/types.divergences.tsv", "D1", "js", 2, lambda f: "own defect"),
           set_field("testdata/snapshots/types.divergences.tsv", "D1", "js", 3, lambda f: "true"),
           set_field("testdata/snapshots/types.divergences.tsv", "D1", "js", 4, lambda f: "see docs/plans/001-decimal-division.md")),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("a types verdict whose evidence is not the oracle's",
     set_field("testdata/snapshots/types.divergences.tsv", "D1", "js", 4, lambda f: f[4].replace("is not of Time", "is not of Date")),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle wrong where no divergence reaches",
     replace("engine_types_oracle_test.go", "func derivesFrom(a, b cqlType) bool {\n",
             "func derivesFrom(a, b cqlType) bool {\n\tif a.name == \"Time\" && b.name == \"Date\" {\n\t\treturn true\n\t}\n"),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleAgreesWhereAllAgree", "."], "the oracle expects"),
    ("the types oracle blaming a reference the harness cannot read",
     replace("engine_types_oracle_test.go", "unencoded.MatchString(d.Reference)", "false"),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("a types verdict made deliberate to escape the oracle",
     edits(set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 2, lambda f: "deliberate"),
           set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 3, lambda f: f[6]),
           set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 4, lambda f: "see docs/plans/003-clasificar-divergencias.md")),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("a reference error that names Unencoded taken for an unreadable answer",
     set_field("testdata/snapshots/types.divergences.tsv", "D223", "js", 5, lambda f: "error: Cannot process Unencoded: input"),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the types oracle reading no definition",
     replace("engine_types_oracle_test.go", "`(?m)^define (D\\d+): (.*)$`", "`(?m)^define (X\\d+): (.*)$`"),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleReadsTheWholeCorpus", "."], "definitions of types.cql"),
    ("an undecided divergence that already expects an answer",
     set_field("testdata/snapshots/scope.divergences.tsv", "D1", "js", 3, lambda f: "Integer 1"),
     CONSISTENT, "expects nothing yet"),
]


# Each command runs without the GITHUB_* variables of the environment: in a
# pull request CI sets GITHUB_BASE_REF, and debt-only-shrinks.sh would compare
# a copy that has no remote with its base. A mutation that needs one sets it.
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


def apply(copy, edits):
    if callable(edits):
        try:
            edits(copy)
        except SystemExit as e:
            return str(e)
        return None
    for path, old, new in edits:
        target = os.path.join(copy, path)
        if old == WRITE:
            os.makedirs(os.path.dirname(target), exist_ok=True)
            open(target, "w").write(new)
        elif old == DELETE:
            os.remove(target)
        elif old == APPEND:
            open(target, "a").write(new)
        else:
            text = open(target).read()
            if text.count(old) != 1:
                return f"{path}: {old!r} found {text.count(old)} times"
            open(target, "w").write(text.replace(old, new))
    return None


# Every command passes on an unmutated copy — prepared by its control, for a
# mutation that needs one — or the mutations prove nothing.
plain = [m for m in mutations if len(m) == 4]
baseline = fresh_copy()
for command in {tuple(m[2]) for m in plain}:
    r = subprocess.run(list(command), cwd=baseline, capture_output=True, text=True, env=CLEAN_ENV)
    if r.returncode != 0:
        print(f"BASELINE FAILS: {' '.join(command)}\n{r.stdout[-2000:]}{r.stderr[-2000:]}")
        sys.exit(2)
for name, _, command, _, control in [m for m in mutations if len(m) == 5]:
    copy = fresh_copy()
    control(copy)
    r = subprocess.run(command, cwd=copy, capture_output=True, text=True, env=CLEAN_ENV)
    if r.returncode != 0:
        print(f"BASELINE FAILS: the control of {name}\n{r.stdout[-2000:]}{r.stderr[-2000:]}")
        sys.exit(2)
print("baseline  passes")

survived = 0
for name, edits, command, text, *_ in mutations:
    copy = fresh_copy()
    problem = apply(copy, edits)
    if problem:
        print(f"NOT APPLIED  {name}: {problem}")
        survived += 1
        continue
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
