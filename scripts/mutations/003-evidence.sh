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
TYPES_VERDICTS = ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."]
FHIR_VERDICTS = ["go", "test", "-count=1", "-run", "TestTheFHIRVerdictsFollowTheOracle", "."]
ANSWERS = "internal/referenceharness/answers/answers.go"
ANSWERS_TESTS = ["go", "test", "-count=1", "./internal/referenceharness/answers"]
ARITHMETIC_VERDICTS = ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."]
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


# No divergence is undecided since PR 6: the mutations of an undecided row
# make one of D380 js, with the two readings it had before ADR 003 decided it,
# and their control checks that the row passes as it is made.
D380_READINGS = ("reading 1: Decimal 65499, as the reference answers [source: testdata/snapshots/arithmetic.js.json] | "
                 "reading 2: Decimal 65500, as this engine answers [source: testdata/snapshots/arithmetic.answers.tsv]; "
                 "the arithmetic oracle (engine_arithmetic_oracle_test.go) does not decide: a truncated quotient at a whole number")


def undecided_d380(copy):
    for column, value in ((2, "undecided"), (3, ""), (4, D380_READINGS)):
        set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", column, lambda f, v=value: v)(copy)


def json_result(path, name, value):
    # Rewrites one definition's answer in a reference file.
    def edit(copy):
        import json
        target = os.path.join(copy, path)
        doc = json.load(open(target))
        doc["results"][name] = value(doc["results"][name])
        json.dump(doc, open(target, "w"), indent=2)
    return edit


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
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 4, lambda f: "unknown")),
     CONSISTENT, "records its two readings", undecided_d380),
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
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/arithmetic.answers.tsv]", "[source: ]"))),
     CONSISTENT, "records its two readings", undecided_d380),
    ("a reading whose source does not exist",
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/arithmetic.answers.tsv]", "[source: testdata/snapshots/nothing.tsv]"))),
     CONSISTENT, "which is not in the repository", undecided_d380),
    ("readings delimited by parentheses inside the answers",
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 4,
               lambda f: 'reading 1: "x(y)" (); reading 2: "a(b)" ()')),
     CONSISTENT, "records its two readings", undecided_d380),
    ("a deliberate difference citing a URL that holds a plan's name",
     set_field(DIVISION, "D11", "java", 4,
               lambda f: f[4].replace("(docs/plans/001-decimal-division.md)", "(https://example.invalid/docs/plans/001-decimal-division.md)")),
     CONSISTENT, "by its whole name"),
    ("a deliberate difference citing a plan that does not decide it",
     set_field(DIVISION, "D11", "java", 4,
               lambda f: f[4].replace("docs/plans/001-decimal-division.md", "docs/plans/002-adr-semantica-cql.md")),
     CONSISTENT, "which does not name D11"),
    ("a reading whose source is a file outside the corpora",
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 4,
               lambda f: f[4].replace("[source: testdata/snapshots/arithmetic.answers.tsv]", "[source: go.mod]"))),
     CONSISTENT, "which is not a file of testdata/snapshots", undecided_d380),
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
    ("ADR 003: null is of every type",
     replace("engine_types_oracle_test.go", 'return typesVerdict{expected: "false", decided: true,', 'return typesVerdict{expected: "true", decided: true,'),
     TYPES_VERDICTS, "the oracle gives"),
    ("ADR 003: null is T is null",
     replace("engine_types_oracle_test.go", 'return typesVerdict{expected: "false", decided: true,', 'return typesVerdict{expected: "null", decided: true,'),
     TYPES_VERDICTS, "the oracle gives"),
    ("ADR 003: a reading of null taken for a reference defect",
     replace("engine_types_oracle_test.go", 'if expectedMatches(d.Expected, d.Engine) && v.decided && !strings.HasPrefix(d.Reference, "error") {',
             'if expectedMatches(d.Expected, d.Engine) && false {'),
     TYPES_VERDICTS, "the oracle gives"),
    ("ADR 003: the Java engine's error on is Any taken for a reading",
     replace("engine_types_oracle_test.go", 'if expectedMatches(d.Expected, d.Engine) && v.decided && !strings.HasPrefix(d.Reference, "error") {',
             'if expectedMatches(d.Expected, d.Engine) && v.decided {'),
     TYPES_VERDICTS, "the oracle gives"),
    ("ADR 003: a fhir null is of every type",
     replace("engine_fhir_oracle_test.go", 'return fhirVerdict{expected: "System.Boolean false", decided: true, unrelated: unrelated,',
             'return fhirVerdict{expected: "System.Boolean true", decided: true, unrelated: unrelated,'),
     FHIR_VERDICTS, "the oracle gives"),
    ("ADR 003: a cast of null fails",
     replace("engine_fhir_oracle_test.go", 'return fhirVerdict{expected: "null", decided: true, unrelated: unrelated,',
             'return fhirVerdict{expected: runTimeError, decided: true, unrelated: unrelated,'),
     FHIR_VERDICTS, "the oracle gives"),
    ("ADR 003: an aggregate taking its signature by the static type",
     replace("engine_fhir_oracle_test.go", "if sig.cost >= elementwise {", "if false && sig.cost >= elementwise {"),
     FHIR_VERDICTS, "the oracle gives"),
    ("ADR 003: an element that reaches no signature fails",
     replace("engine_fhir_oracle_test.go", 'r := c.operateOnNull(op, reason+": no signature takes a "+v.runtime.String())',
             'r := fhirVerdict{expected: runTimeError, reason: reason}'),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree", "."], "the oracle expects"),
    ("ADR 003: two conversions in a row refused",
     replace("engine_fhir_oracle_test.go", 'reason += " through two conversions in a row (" + undecidedADR + ", decision 4)"',
             'return fhirVerdict{expected: translationError, reason: reason}, nil'),
     FHIR_VERDICTS, "the oracle gives"),
    ("ADR 003: the Java engine's run-time error taken for a reading",
     replace("engine_fhir_oracle_test.go", "case v.decided && refText != runTimeError:", "case v.decided:"),
     FHIR_VERDICTS, "the oracle gives"),
    ("ADR 003: the exact reading widened as the specification's",
     replace("engine_arithmetic_oracle_test.go", "\tif r.exact {\n\t\tif c.rounded {", "\tif false && r.exact {\n\t\tif c.rounded {"),
     ARITHMETIC_VERDICTS, "the oracle gives"),
    ("ADR 003: the exact reading floored at the scale",
     replace("engine_arithmetic_oracle_test.go", "v.exactReading = exactly.conform(n)", "v.exactReading = aConform{typ: v.policy.typ, lo: v.policy.v, hi: v.policy.v}"),
     ARITHMETIC_VERDICTS, "the oracle gives"),
    ("ADR 003: the exact reading without its margin",
     replace("engine_arithmetic_oracle_test.go", "return decimal.Max(d.Abs(), decimal.NewFromInt(1)).Mul(oracleMargin)", "return decimal.Zero"),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want undecided"),
    ("ADR 003: a reference off the exact reading not taken for a defect",
     replace("engine_arithmetic_oracle_test.go", "case isExact(engine) && !isExact(reference):", "case isExact(engine) && isExact(reference):"),
     ARITHMETIC_VERDICTS, "the oracle gives"),
    ("ADR 003: this engine off the policy where the exact reading decides",
     replace("testdata/snapshots/arithmetic.answers.tsv", "D380\tDecimal 65500", "D380\tDecimal 65501"),
     ARITHMETIC_VERDICTS, "the oracle gives"),
    ("ADR 003: a types cast of null fails",
     replace("engine_types_oracle_test.go", 'return typesVerdict{expected: "null", decided: true,', 'return typesVerdict{expected: "error", decided: true,'),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleAgreesWhereAllAgree", "."], "the oracle expects"),
    ("ADR 003: a types cast of null left undecided",
     replace("engine_types_oracle_test.go", 'return typesVerdict{expected: "null", decided: true,', 'return typesVerdict{undecided: true,'),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleAgreesWhereAllAgree", "."], "leaves undecided what it should decide"),
    ("ADR 003: a rounded quotient taken for an exact one",
     replace("engine_arithmetic_oracle_test.go", "rounded = rounded || !q.Mul(y).Equal(x)", "rounded = rounded || false && !q.Mul(y).Equal(x)"),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want undecided"),
    ("ADR 003: a quotient truncated without its margin",
     replace("engine_arithmetic_oracle_test.go", "\tif r.exact {\n\t\tq = r.widen(q)\n\t}\n", ""),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want undecided"),
    ("ADR 003: the exact reading taking exp past where it computes",
     replace("engine_arithmetic_oracle_test.go", "if r.exact && e.lo.LessThan(expFloor) {", "if false && r.exact && e.lo.LessThan(expFloor) {"),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want undecided"),
    ("ADR 003: a logarithm bounded after it is divided",
     replace("engine_arithmetic_oracle_test.go", "rounded: true}\n\tif r.exact {\n\t\tc = r.widen(c)\n\t}\n", "rounded: true}\n"),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want undecided"),
    ("ADR 003: Round by a division at 16 digits",
     replace("engine_arithmetic_oracle_test.go", "return m.Shift(-places) // exactly", "return m.Div(shift) //"),
     ["go", "test", "-count=1", "-run", "TestTheExactReadingHoldsItsControls", "."], "want Decimal 0"),
    ("ADR 003 not naming a row it decides",
     replace("docs/adr/003-lo-que-la-especificacion-no-decide.md", "D129 D172 D265", "D172 D265"),
     ["go", "test", "-count=1", "-run", "TestTheUndecidedADRNamesItsRows", "."], "names other types rows"),
    ("the types oracle swapping the classes",
     replace("engine_types_oracle_test.go", "\tif expectedMatches(d.Expected, d.Engine) {\n\t\td.Class = referenceDefect",
             "\tif !expectedMatches(d.Expected, d.Engine) {\n\t\td.Class = referenceDefect"),
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
    ("a types verdict made deliberate to escape the oracle",
     edits(set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 2, lambda f: "deliberate"),
           set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 3, lambda f: f[6]),
           set_field("testdata/snapshots/types.divergences.tsv", "D13", "java", 4, lambda f: "see docs/plans/003-clasificar-divergencias.md")),
     ["go", "test", "-count=1", "-run", "TestTheTypesVerdictsFollowTheOracle", "."], "the oracle gives"),
    # PR 7: what the harness writes now that it writes everything.
    ("a Number of no type matched whatever its value",
     replace(ANSWERS, "return leaf(Answer{Kind: \"number\", Text: typ + \" \" + a.Num.String(), Num: a.Num}, b)",
             "return leaf(Answer{Kind: \"number\", Text: typ + \" \" + b.Num.String(), Num: b.Num}, b)"),
     ANSWERS_TESTS, "a Number of another value"),
    ("an Object matched whatever its fields are named",
     replace(ANSWERS, "if !slices.Equal(a.Names, b.Names) || len(a.Parts) != len(b.Parts) {", "if len(a.Parts) != len(b.Parts) {"),
     ANSWERS_TESTS, "an Object with a field renamed"),
    ("an interval of no type matched however it is closed",
     replace(ANSWERS, "return a.Text[len(\"Interval\")] == b.Text[len(\"Interval\")] && a.Text[len(a.Text)-1] == b.Text[len(b.Text)-1] &&",
             "return true &&"),
     ANSWERS_TESTS, "closed otherwise"),
    ("a Code's null field read as an empty one",
     replace(ANSWERS, "if json.Unmarshal(raw, &s) != nil || s == nil {\n\t\treturn Answer{Kind: \"null\", Text: \"null\"}",
             "if json.Unmarshal(raw, &s) != nil || s == nil {\n\t\treturn Answer{Kind: \"string\", Text: `\"\"`}"),
     ANSWERS_TESTS, "null fields against those this engine leaves unset"),
    ("a Code of another system in the Java reference",
     json_result("testdata/snapshots/types.java.json", "D690", lambda v: {**v, "system": "s"}), REFS, "new: D690"),
    ("a Concept short of its code in cql-execution's reference",
     json_result("testdata/snapshots/types.js.json", "D44", lambda v: {**v, "codes": []}), REFS, "new: D44"),
    ("a reference value the harness writes as Unencoded again",
     json_result("testdata/snapshots/types.js.json", "D393", lambda v: {"type": "Unencoded:Object", "value": "[object Object]"}),
     ["go", "test", "-count=1", "-run", "TestTheReferencesHoldNothingUnencoded", "."], "could not encode"),
    ("a regenerated reference that moves a value that was encoded",
     json_result("testdata/snapshots/types.js.json", "D168", lambda v: {**v, "value": "b"}), REFS, "new: D168"),
    ("the types oracle expecting another tuple where a match has no type",
     replace("engine_types_oracle_test.go", "answer: \"Tuple{a: 1}\"", "answer: \"Tuple{a: 2}\""),
     REFS, "no oracle confirms the type"),
    ("the fhir oracle expecting a tuple where a match has no type",
     replace("engine_fhir_oracle_test.go", 'value: "System.ValueSet {id: System.String "', 'value: "System.Tuple {id: System.String "'),
     REFS, "no oracle confirms the type"),
    ("an Object matched by a value that is no structure",
     replace(ANSWERS, "return b.Kind == \"structure\" && sameParts(a, b, leaf)", "return sameParts(a, b, leaf)"),
     ANSWERS_TESTS, "an empty Object is no empty list"),
    ("an untyped match never checked",
     replace(ORACLE, "if answers.HoldsUntyped(reference) {", "if false {"), REFS, "checks nothing"),
    ("the types oracle reading no definition",
     replace("engine_types_oracle_test.go", "`(?m)^define (D\\d+): (.*)$`", "`(?m)^define (X\\d+): (.*)$`"),
     ["go", "test", "-count=1", "-run", "TestTheTypesOracleReadsTheWholeCorpus", "."], "definitions of types.cql"),
    ("the fhir oracle ignoring the bases of a FHIR type",
     replace("engine_fhir_oracle_test.go", "for name := a.name; name != \"\"; name = m.base[name] {",
             "for name := a.name; name != \"\"; name = \"\" {"),
     ["go", "test", "-count=1", "-run", "TestTheFHIRVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the fhir oracle looking an unqualified name up in System first",
     replace("engine_fhir_oracle_test.go", "\tif m.base[\"FHIR.\"+s] != \"\" {",
             "\tif slices.Contains(systemTypes, s) {\n\t\treturn fNamed(\"System.\" + s), nil\n\t}\n\tif m.base[\"FHIR.\"+s] != \"\" {"),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleTypesStaticallyAsJava", "."], "the Java translator refuses it"),
    ("the fhir oracle taking a choice's last type, not the data's branch",
     replace("engine_fhir_oracle_test.go", "node, typ = v, choice.name", "node, typ = v, spec.elem[len(spec.elem)-1].name"),
     ["go", "test", "-count=1", "-run", "TestTheFHIRVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the fhir oracle ignoring Conversion Precedence",
     replace("engine_fhir_oracle_test.go", "func conversionCost(to fType) int {\n", "func conversionCost(to fType) int {\n\treturn toSimpleType\n"),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree", "."], "the oracle expects"),
    ("the fhir oracle building an instance with a corrupt value",
     replace("engine_fhir_oracle_test.go", "v = strings.Trim(value, \"'\")", "v = \"CORRUPT\""),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleValuesAreTheReferences", "."], "the oracle expects"),
    ("the fhir oracle reading no definition",
     replace("engine_fhir_oracle_test.go", "`(?m)^define (D\\d+): (.*)$`", "`(?m)^define (X\\d+): (.*)$`"),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleReadsTheWholeCorpus", "."], "definitions of fhir.cql"),
    ("the fhir decision naming one cast less",
     replace("docs/plans/003-clasificar-divergencias.md", "begin -->\nD0 D100 ", "begin -->\nD100 "),
     ["go", "test", "-count=1", "-run", "TestTheFHIRDecisionNamesItsCasts", "."], "names other casts"),
    ("a fhir own defect made deliberate",
     edits(set_field("testdata/snapshots/fhir.divergences.tsv", "D16", "java", 2, lambda f: "deliberate"),
           set_field("testdata/snapshots/fhir.divergences.tsv", "D16", "java", 3, lambda f: f[6]),
           set_field("testdata/snapshots/fhir.divergences.tsv", "D16", "java", 4, lambda f: "see docs/plans/003-clasificar-divergencias.md D16")),
     ["go", "test", "-count=1", "-run", "TestTheFHIRVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the fhir oracle taking a reference's rendering for its value",
     replace("engine_fhir_oracle_test.go", '"java FHIR.id Patient/p": ', '"java FHIR.id Patient/q": '),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleValuesAreTheReferences", "."], "the oracle expects"),
    ("an engine answer hidden by the comparison that is not the known one",
     replace("engine_fhir_oracle_test.go", '"D794": "FHIR primitive of System.String abc"', '"D794": "FHIR primitive of System.String abd"'),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree", "."], "the known ones"),
    ("this engine's Booleans rendered negated for the fhir oracle",
     replace("engine_fhir_oracle_test.go", 'return fmt.Sprintf("System.Boolean %v", n.Bool())', 'return fmt.Sprintf("System.Boolean %v", !n.Bool())'),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleAgreesWithTheReferencesWhereAllAgree", "."], "the known ones"),
    ("the Java engine's answer to an as missing",
     replace("testdata/snapshots/fhir.java.json", '"D19" : ', '"D19x" : '),
     ["go", "test", "-count=1", "-run", "TestTheFHIROracleTypesStaticallyAsJava", "."], "gives no answer"),
    ("the scale policy rounding instead of flooring",
     replace("engine_arithmetic_oracle_test.go", "return d.RoundFloor(8)", "return d.Round(8)"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the scale policy flooring a difference",
     replace("engine_arithmetic_oracle_test.go", "return pRange(aPolicy{typ: typ, v: a.v.Sub(b.v), unscaled: a.unscaled || b.unscaled})", "return scale(a.v.Sub(b.v))"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the arithmetic oracle's division by 0 not null",
     replace("engine_arithmetic_oracle_test.go", "case b.exact() && b.lo.IsZero():\n\t\treturn cNull(typ)", "case b.exact() && b.lo.IsZero():\n\t\treturn aConform{typ: typ}"),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the arithmetic oracle reading a literal corrupt",
     replace("engine_arithmetic_oracle_test.go", "p.pos += len(lit)\n", "p.pos += len(lit)\n\t\tlit += \"1\"\n"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("the arithmetic oracle allowing a far wider binary64 error",
     replace("engine_arithmetic_oracle_test.go", 'decimal.RequireFromString("0.0000000000000004")', 'decimal.RequireFromString("0.0001")'),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the arithmetic oracle allowing no rounding at the scale",
     replace("engine_arithmetic_oracle_test.go", "\t\t\ts = scaleStep\n", "\t\t\ts = decimal.Zero\n"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyIsConformant", "."], "the policy answers"),
    ("the arithmetic oracle's div Euclidean",
     replace("engine_arithmetic_oracle_test.go", "return decimal.NewFromBigInt(new(big.Int).Quo(r.Num(), r.Denom()), 0)", "return decimal.NewFromBigInt(new(big.Int).Div(r.Num(), r.Denom()), 0)"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("the arithmetic oracle rounding half up",
     replace("engine_arithmetic_oracle_test.go", "m := d.Abs().Mul(shift).Add(half).Floor()", "m := d.Mul(shift).Add(half).Floor().Abs()"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the arithmetic oracle keeping the nulls of an aggregate",
     replace("engine_arithmetic_oracle_test.go", "\t\tif c.null {\n\t\t\tcontinue\n\t\t}\n", ""),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("a cql-execution answer a micro off its value",
     replace("testdata/snapshots/arithmetic.js.json", '"0.8555992052357225"', '"0.8555982052357225"'),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("an exception where all agree that no longer is one",
     replace("engine_arithmetic_oracle_test.go", '"D114 java": failsOnInfinity', '"D114x java": failsOnInfinity'),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticOracleAllowsWhatAllAgreeOn", "."], "as this engine does"),
    ("this engine off the policy where no divergence reaches it",
     replace("testdata/snapshots/arithmetic.answers.tsv", "D2\tDecimal 0.0081", "D2\tDecimal 0.0082"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("a Java answer of a sign no rounding gives",
     replace("testdata/snapshots/arithmetic.java.json", '"value" : "0.000000001"', '"value" : "-0.000000001"'),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("the arithmetic oracle's capped power losing its sign",
     replace("engine_arithmetic_oracle_test.go", "p = p.Neg()", "p = p.Abs()"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the scale policy nulling Round of a null precision",
     replace("engine_arithmetic_oracle_test.go", "\t\targs = args[:1]\n", ""),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the arithmetic oracle adding Integers before converting them",
     replace("engine_arithmetic_oracle_test.go", "adds Decimals.\n\tfor i := range items {\n\t\titems[i].typ = typ\n\t}\n", "adds Decimals.\n"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("the arithmetic oracle's exponential underflowing to an exact 0",
     replace("engine_arithmetic_oracle_test.go", "return decimal.New(1, -60)", "return decimal.Zero"),
     ["go", "test", "-count=1", "-run", "TestTheScalePolicyHoldsItsControls", "."], "ADR 002 gives"),
    ("this engine an eighth-place step off an exact root",
     replace("testdata/snapshots/arithmetic.answers.tsv", "D191\tDecimal 0.5\n", "D191\tDecimal 0.50000001\n"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("this engine a step off a floored answer",
     replace("testdata/snapshots/arithmetic.answers.tsv", "D116\tDecimal 0\n", "D116\tDecimal 2.5\n"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("this engine a float a digit off an exact Exp",
     replace("testdata/snapshots/arithmetic.answers.tsv", "D27\tDecimal 1\n", "D27\tDecimal 1.0000000000000002\n"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("this engine's fractional power a float off, within the snapshot's tolerance",
     replace("funcs/math_ops.go", "FloatScaled(math.Pow(fv, fe))", "FloatScaled(math.Nextafter(math.Pow(fv, fe), math.Inf(-1)))"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScalePolicy", "."], "this engine answers"),
    ("the arithmetic decision naming one row less",
     replace("docs/plans/003-clasificar-divergencias.md", "<!-- arithmetic decision 1, written by -classify-arithmetic: begin -->\nD10 ", "<!-- arithmetic decision 1, written by -classify-arithmetic: begin -->\n"),
     ["go", "test", "-count=1", "-run", "TestTheArithmeticDecisionNamesItsRows", "."], "names other rows"),
    ("the scope oracle evaluating a definition in its caller's scope",
     replace('engine_scope_oracle_test.go', "\tif l.dynamic {", "\tif true {"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ("the scope oracle's return not distinct",
     replace('engine_scope_oracle_test.go', "if !slices.ContainsFunc(out.list, func(o sValue) bool { return o.kind == r.kind && o.i == r.i }) {", "if true {"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ("the scope oracle's where keeping a null condition",
     replace('engine_scope_oracle_test.go', "if c.kind != 'b' || c.i != 1 {", "if c.kind == 'b' && c.i != 1 {"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ("the scope oracle's aggregate of no non-null element 0",
     replace('engine_scope_oracle_test.go', "return sNull // \"If the source contains no non-null elements, null is returned\"", "return scopeInteger(0)"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ('the scope oracle counting the null elements',
     replace('engine_scope_oracle_test.go', "if e.kind == 'i' {\n\t\t\tvalues = append(values, e.i) // the non-null elements", "if e.kind != 'l' {\n\t\t\tvalues = append(values, e.i) // the non-null elements"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ("the scope oracle's Integer past 32 bits",
     replace('engine_scope_oracle_test.go', "if i < math.MinInt32 || i > math.MaxInt32 {", "if i < math.MinInt32 && false {"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ('the scope oracle binding the alias before its source',
     replace('engine_scope_oracle_test.go', "source, err := l.eval(q.source, env)", "source, err := l.eval(q.source, env.with(q.alias, scopeInteger(5)))"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ('the scope oracle binding a let before its value',
     replace('engine_scope_oracle_test.go', "v, err := l.eval(q.let, inner)", "v, err := l.eval(q.let, inner.with(q.letName, scopeInteger(0)))"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ("the scope oracle keeping every null of a return",
     replace("engine_scope_oracle_test.go", "o.kind == r.kind && o.i == r.i }", "o.kind == r.kind && o.i == r.i && r.kind != 'n' }"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ("the scope oracle's let leaving an outer alias of its name in place",
     replace("engine_scope_oracle_test.go", "\t\t\tinner = inner.with(q.letName, v)", "\t\t\tif _, bound := inner.lookup(q.letName); !bound {\n\t\t\t\tinner = inner.with(q.letName, v)\n\t\t\t}"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ("the scope oracle's Count of a null list null",
     replace("engine_scope_oracle_test.go", "return scopeInteger(0) // \"If the list is null, the result is 0\"", "return sNull"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleHoldsItsControls", "."], "the specification gives"),
    ('the scope preamble read wrong',
     replace('testdata/snapshots/scope.cql', "define DB: B * 2\n", "define DB: B * 3\n"),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ('a Java answer of the scope area corrupt',
     replace('testdata/snapshots/scope.java.json', "\"D4\" : {\n      \"type\" : \"Integer\",\n      \"value\" : \"100\"", "\"D4\" : {\n      \"type\" : \"Integer\",\n      \"value\" : \"101\""),
     ["go", "test", "-count=1", "-run", "TestTheScopeOracleAgreesWithJava", "."], "the oracle gives"),
    ("this engine evaluating a definition in its caller's scope",
     replace('eval/evaluator.go', "if scope := e.ctx.definitionScope(); scope != e.ctx {", "if scope := e.ctx.definitionScope(); false && scope != e.ctx {"),
     ["go", "test", "-count=1", "-run", "TestTheEngineFollowsTheScopeOracle", "."], "this engine answers"),
    ('a scope verdict changed',
     replace('testdata/snapshots/scope.divergences.tsv', "D1\tjs\treference defect\t", "D1\tjs\tdeliberate\t"),
     ["go", "test", "-count=1", "-run", "TestTheScopeVerdictsFollowTheOracle", "."], "the oracle gives"),
    ("a translator's refusal read as a run-time error",
     replace("internal/referenceharness/answers/answers.go", "if v.Error == \"Translation\" {", "if false {"),
     ["go", "test", "-count=1", "-run", "TestAnswersMatchTheReferences", "."], "gone: D"),
    ("an undecided divergence that already expects an answer",
     edits(undecided_d380, set_field("testdata/snapshots/arithmetic.divergences.tsv", "D380", "js", 3, lambda f: "Decimal 1")),
     CONSISTENT, "expects nothing yet", undecided_d380),
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
