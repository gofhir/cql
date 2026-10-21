#!/bin/sh
# Mutations of the fields of Code and Concept, increment 004 PRs 1 and 2
# (docs/plans/004-campos-de-code.md). Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/004-code-fields.sh [work-directory]
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
CONTEXT = "eval/context.go"
TYPES = "types/clinical.go"
FUNCS = "funcs/type_ops.go"
ANSWERS = "internal/referenceharness/answers/answers.go"
SNAPSHOT = "engine_snapshot_test.go"
ORACLE = "engine_codes_oracle_test.go"
CORPUS = "testdata/snapshots/codes.cql"
SETS_CORPUS = "testdata/snapshots/codesets.cql"
MODE = "funcs/list_advanced.go"
PARSER_GO = "conformance/output_parser.go"


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


# What the engine answers: the cases written by hand and the differential.
ENGINE = run("TestCodeFieldsKeepNullApartFromEmpty", "TestCodeOperationsTellCodesApart", "TestAnswersMatchTheReferences")
PARSER = ["go", "test", "-count=1", "-run", "TestParseExpectedOutput", "./conformance"]
SNAPSHOTS = run("TestAnswersMatchTheSnapshots")
# What the oracle computes: the verdicts, where all agree, and the whole corpus.
ORACLE_TESTS = run("TestTheCodesVerdictsFollowTheOracle", "TestTheCodesOracleAgreesWhereAllAgree",
                   "TestTheCodesOracleReadsTheWholeCorpus")

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The engine.
    ("A4: the equality of two elements compares their addresses",
     (EVAL, "\tcase *a == *b:\n\t\treturn elementsEqual", "\tcase a == b:\n\t\treturn elementsEqual"), ENGINE, "FAIL"),
    ("a declared code loses its code system's version",
     (CONTEXT, "return cs.System, cs.Version", "return cs.System, nil"), ENGINE, "FAIL"),
    ("A7: a Code inside a container compares by the Boolean Equal",
     (EVAL, "\t\treturn codeVerdict(l, r), true", "\t\tif l.Equal(r) {\n\t\t\treturn elementsEqual, true\n\t\t}\n\t\treturn elementsUnequal, true"),
     ENGINE, "FAIL"),
    ("the order of a Concept's codes is ignored",
     (EVAL, "verdicts[i] = codeVerdict(a.Codes[i], b.Codes[i])", "verdicts[i] = codeVerdict(a.Codes[i], b.Codes[len(b.Codes)-1-i])"),
     ENGINE, "FAIL"),
    ("ToConcept copies the Code's display",
     (FUNCS, "\t\t// null (the Java engine 3.26.0 agrees).\n\t\treturn cqltypes.NewConcept([]cqltypes.Code{c}, nil), nil",
      "\t\t// null (the Java engine 3.26.0 agrees).\n\t\treturn cqltypes.NewConcept([]cqltypes.Code{c}, c.Display), nil"),
     ENGINE, "FAIL"),
    ("equality decides at the first unknown element, not by the conjunction",
     (EVAL, "\t\tcase elementsUnknown:\n\t\t\tresult = elementsUnknown", "\t\tcase elementsUnknown:\n\t\t\treturn elementsUnknown"),
     ENGINE, "FAIL"),
    ("equality ignores the display",
     (EVAL, "\t\telementTextVerdict(a.Display, b.Display),\n\t\telementTextVerdict(a.System, b.System),",
      "\t\telementsEqual,\n\t\telementTextVerdict(a.System, b.System),"), ENGINE, "FAIL"),
    ("equality ignores the version",
     (EVAL, "\t\telementTextVerdict(a.Version, b.Version),\n\t)", "\t\telementsEqual,\n\t)"), ENGINE, "FAIL"),
    ("equality is false where an element is null on one side",
     (EVAL, "\tcase a == nil || b == nil:\n\t\treturn elementsUnknown\n\tcase *a == *b:",
      "\tcase a == nil || b == nil:\n\t\treturn elementsUnequal\n\tcase *a == *b:"), ENGINE, "FAIL"),
    ("equality ignores a Concept's display",
     (EVAL, "return conjunction(codes, elementTextVerdict(a.Display, b.Display))", "return codes"), ENGINE, "FAIL"),
    ("equivalence looks at the display",
     (TYPES, "return sameTextFold(c.System, o.System) && sameTextFold(c.Code, o.Code)",
      "return sameTextFold(c.System, o.System) && sameTextFold(c.Code, o.Code) && sameTextFold(c.Display, o.Display)"),
     ENGINE, "FAIL"),
    ("equivalence takes the empty String for null",
     (TYPES, "func sameTextFold(a, b *string) bool {\n\tif a == nil || b == nil {",
      "func sameTextFold(a, b *string) bool {\n\tif a != nil && *a == \"\" {\n\t\ta = nil\n\t}\n\tif b != nil && *b == \"\" {\n\t\tb = nil\n\t}\n\tif a == nil || b == nil {"),
     ENGINE, "FAIL"),
    ("reading a field takes the empty String for null",
     (EVAL, "func optionalElement(s *string) fptypes.Value {\n\tif s == nil {",
      "func optionalElement(s *string) fptypes.Value {\n\tif s == nil || *s == \"\" {"), ENGINE, "FAIL"),
    ("the selector takes the empty String where an element was not set",
     (EVAL, "\tv, ok := elements[name]\n\tif !ok || v == nil {\n\t\treturn nil\n\t}",
      "\tv, ok := elements[name]\n\tif !ok || v == nil {\n\t\treturn cqltypes.Str(\"\")\n\t}"), ENGINE, "FAIL"),
    ("a Code selector names its code system with the quotes",
     ("compiler/builder.go", "\t\tce.System = undelimitedIdentifier(csid.Identifier(), csid)", "\t\tce.System = csid.GetText()"),
     ENGINE, "FAIL"),
    ("a Code selector loses its code system's version",
     (EVAL, "\tsystem, version := e.ctx.codeSystemOf(n.System)\n", "\tsystem, version := e.ctx.codeSystemOf(n.System)\n\tversion = nil\n"),
     ENGINE, "FAIL"),
    ("a Code selector loses its display",
     (EVAL, "\treturn cqltypes.NewCode(system, cqltypes.Str(n.Code), n.Display, version), nil",
      "\treturn cqltypes.NewCode(system, cqltypes.Str(n.Code), nil, version), nil"), ENGINE, "FAIL"),
    ("ELM JSON reads an empty display as null",
     ("elm/json.go", "\te.Display = raw.Display\n", "\te.Display = raw.Display\n\tif e.Display != nil && *e.Display == \"\" {\n\t\te.Display = nil\n\t}\n"),
     ["go", "test", "-count=1", "-run", "TestCodeFieldsSurviveTheRoundTrip", "./elm"], "FAIL"),
    ("an expansion's empty element is a String",
     ("eval/terminology_expand.go", "\tif s == \"\" {\n\t\treturn nil\n\t}\n\treturn &s", "\treturn &s"),
     ["go", "test", "-count=1", "-run", "TestExpandValueSet_EmptyIsNull", "./eval"], "FAIL"),
    # PR 2: the operations that hold values.
    ("A5: the operations compare a Code by system and code only",
     (TYPES, "\treturn SameText(c.System, o.System) && SameText(c.Code, o.Code) &&\n\t\tSameText(c.Display, o.Display) && SameText(c.Version, o.Version)",
      "\treturn SameText(c.System, o.System) && SameText(c.Code, o.Code)"), ENGINE, "FAIL"),
    ("the operations ignore a Code's version",
     (TYPES, "SameText(c.Display, o.Display) && SameText(c.Version, o.Version)", "SameText(c.Display, o.Display)"), ENGINE, "FAIL"),
    ("the operations ignore a Concept's display",
     (TYPES, "if len(c.Codes) != len(o.Codes) || !SameText(c.Display, o.Display) {", "if len(c.Codes) != len(o.Codes) {"), ENGINE, "FAIL"),
    ("Mode keys an Integer by its type, apart from an equal Decimal",
     (MODE, "\t\twriteNumberKey(b, decimal.NewFromInt(x.Value()))", "\t\tb.WriteString(\"Integer:\" + x.String())"), ENGINE, "FAIL"),
    ("Mode counts a Code by its String",
     (MODE, "\tcase cqltypes.Code:\n\t\twriteCodeKey(b, x)", "\tcase cqltypes.Code:\n\t\tb.WriteString(x.String())"), ENGINE, "FAIL"),
    ("Mode keys a null element as the empty String",
     (MODE, "\tif s == nil {\n\t\tb.WriteString(\"-;\")\n\t\treturn\n\t}", "\tif s == nil {\n\t\tb.WriteString(\"\\\"\\\";\")\n\t\treturn\n\t}"), ENGINE, "FAIL"),
    ("Mode keys a tuple by its String",
     (MODE, "\tcase cqltypes.Tuple:\n\t\tnames := make", "\tcase cqltypes.Tuple:\n\t\tb.WriteString(x.String())\n\t\treturn\n\t\tnames := make"),
     ENGINE, "FAIL"),
    ("Mode keys a list by its String",
     (MODE, "\tcase cqltypes.List:\n\t\tb.WriteByte('[')", "\tcase cqltypes.List:\n\t\tb.WriteString(x.String())\n\t\treturn\n\t\tb.WriteByte('[')"),
     ENGINE, "FAIL"),
    ("A11: the conformance parser reads null as the text null",
     (PARSER_GO, "\tif s == \"null\" {\n\t\treturn nil\n\t}", ""), PARSER, "FAIL"),
    ("A11: the conformance parser ignores the version",
     (PARSER_GO, "\t\tcase \"version\":\n\t\t\tversion = value\n", ""), PARSER, "FAIL"),
    # The harness: how an answer is written.
    ("the harness writes the empty String as null",
     (ANSWERS, "func stringField(s *string) Answer {\n\tif s == nil {", "func stringField(s *string) Answer {\n\tif s == nil || *s == \"\" {"),
     ENGINE, "FAIL"),
    ("the snapshot writes null as the empty String",
     (SNAPSHOT, "func snapshotElement(s *string) string {\n\tif s == nil {", "func snapshotElement(s *string) string {\n\tif s == nil && false {"),
     SNAPSHOTS, "FAIL"),
    # The oracle.
    ("the oracle: an element null on one side is unequal",
     (ORACLE, "\tcase a == nil || b == nil:\n\t\treturn triNull", "\tcase a == nil || b == nil:\n\t\treturn triFalse"),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: the conjunction stops at the first unknown",
     (ORACLE, "\t\tcase triNull:\n\t\t\tout = triNull", "\t\tcase triNull:\n\t\t\treturn triNull"), ORACLE_TESTS, "the oracle"),
    ("the oracle: equivalence counts case",
     (ORACLE, "return strings.EqualFold(*a, *b)", "return *a == *b"), ORACLE_TESTS, "the oracle"),
    ("the oracle: equivalence compares the display too",
     (ORACLE, "return textEquivalent(a.fields[0], b.fields[0]) && textEquivalent(a.fields[1], b.fields[1])",
      "return textEquivalent(a.fields[0], b.fields[0]) && textEquivalent(a.fields[1], b.fields[1]) && textEquivalent(a.fields[2], b.fields[2])"),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: two null elements of a list are unknown",
     (ORACLE, "\t\tif a[i].kind == \"null\" && b[i].kind == \"null\" {", "\t\tif false {"), ORACLE_TESTS, "the oracle"),
    ("the oracle: a declared display of the empty String reads as none",
     (ORACLE, "\t\tif quoted == \"\" {", "\t\tif quoted == \"\" || quoted == \"''\" {"), ORACLE_TESTS, "the oracle"),
    ("the oracle: a declared code without its code system's version",
     (ORACLE, "fields: [4]*string{&code, cs.id, optional(m[4]), cs.version}", "fields: [4]*string{&code, cs.id, optional(m[4]), nil}"),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: an omitted element reads as the empty String",
     (ORACLE, "func codesText(s *string) codesValue {\n\tif s == nil {\n\t\treturn codesNull\n\t}",
      "func codesText(s *string) codesValue {\n\tif s == nil {\n\t\te := \"\"\n\t\ts = &e\n\t}"), ORACLE_TESTS, "the oracle"),
    ("the oracle: a Code selector without its code system's version",
     (ORACLE, "fields: [4]*string{&code, cs.fields[1], nil, cs.fields[3]}", "fields: [4]*string{&code, cs.fields[1], nil, nil}"),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: an unknown equality is a match",
     (ORACLE, "\treturn codesEqual(a, b) == triTrue\n}", "\treturn codesEqual(a, b) != triFalse\n}"), ORACLE_TESTS, "the oracle"),
    ("the oracle: union keeps the repeated values",
     (ORACLE, "return codesValue{kind: \"list\", codes: distinctCodes(append(slices.Clone(a.codes), b.codes...))}",
      "return codesValue{kind: \"list\", codes: append(slices.Clone(a.codes), b.codes...)}"), ORACLE_TESTS, "the oracle"),
    ("the oracle: intersect and except keep the repeated values",
     (ORACLE, "return codesValue{kind: \"list\", codes: distinctCodes(kept)}", "return codesValue{kind: \"list\", codes: kept}"), ORACLE_TESTS, "the oracle"),
    ("the oracle: a query's return is not distinct",
     (ORACLE, "\t\t\tv = codesValue{kind: \"list\", codes: distinctCodes(v.codes)}\n\t\t\tr.reason = \"a query's", "\t\t\tr.reason = \"a query's"),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: Mode takes the last distinct value",
     (ORACLE, "\tif tie {\n\t\tpanic(", "\tif d := distinctCodes(list); len(d) > 0 {\n\t\tbest = d[len(d)-1]\n\t}\n\tif tie {\n\t\tpanic("),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: IndexOf by equivalence",
     (ORACLE, "\t\tif sameCodesValue(x, v) {\n\t\t\treturn i", "\t\tif codesEquivalent(x, v) {\n\t\t\treturn i"), ORACLE_TESTS, "the oracle"),
    ("the oracle: Count counts the nulls",
     (ORACLE, "\t\t\tif x.kind != \"null\" {\n\t\t\t\tn++", "\t\t\tif x.kind != \"null\" || true {\n\t\t\t\tn++"), ORACLE_TESTS, "the oracle"),
    ("the oracle: IndexOf of a null is -1",
     (ORACLE, "\t\tif v.kind == \"null\" {\n\t\t\tr.reason = \"IndexOf of a null is null\"", "\t\tif false {\n\t\t\tr.reason = \"IndexOf of a null is null\""),
     ORACLE_TESTS, "the oracle"),
    ("the oracle: Mode counts the nulls",
     (ORACLE, "\t\t\tif x.kind != \"null\" {\n\t\t\t\tpresent = append", "\t\t\tif true {\n\t\t\t\tpresent = append"),
     ORACLE_TESTS, "the oracle"),
    ("ADR 003, decision 5: a reference by equivalence taken for a defect",
     (ORACLE, "\tif d.Class == referenceDefect && v.decided && v.alternative != \"\" && d.Reference == v.alternative {",
      "\tif false && d.Class == referenceDefect && v.decided && v.alternative != \"\" && d.Reference == v.alternative {"),
     run("TestTheModeDecisionIsTheOnlyReadingLeftAside"), "FAIL"),
    ("ADR 003, decision 5: the equivalence reading not computed",
     (ORACLE, "\t\t\tif r.mode && r.alternative == \"\" {", "\t\t\tif false && r.mode && r.alternative == \"\" {"),
     run("TestTheModeDecisionIsTheOnlyReadingLeftAside"), "FAIL"),
    ("the codesets corpus is not the one the references answered",
     (SETS_CORPUS, "define D0: ", "define D0: Count({ Code { code: 'a' } }) + "), run("TestAnswersMatchTheReferences"), "FAIL"),
    ("the corpus is not the one the references answered",
     (CORPUS, "define D0: ", "define D0: (Code { code: 'a' }).code = 'a' or "), run("TestAnswersMatchTheReferences"), "FAIL"),
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
