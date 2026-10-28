#!/bin/sh
# Mutations of what plan 010 puts in to read a string as a Quantity
# (docs/plans/010-to-quantity.md): the reader of the format, the number kept
# as written within the Decimal's range and precision, the unit a literal of
# UCUM or a calendar keyword, the numeric overloads, convert and
# ConvertsToQuantity, the oracles written apart from the reader, and the area
# written by hand that holds them. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/010-quantity.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-010.XXXXXX")
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
REFERENCES = run("TestAnswersMatchTheReferences")
CONSISTENT = run("TestTheDivergencesAreConsistent")
FORMS = run("TestTheQuantityFormsAgree")
BUDGET = run("TestAQuantityStringOfMillionsOfBytesIsReadInBoundedResources", package="./funcs/")
GIVEN = run("TestToQuantityReadsWhatItIsGiven", package="./funcs/")
KEYWORDS = run("TestTheCalendarKeywordsAreTheGrammars", package="./types/")
READ = "funcs/quantity_string.go"
NUMBER = "funcs/decimal_scale.go"
ORACLE = "funcs/quantity_string_test.go"
EVAL = "eval/evaluator.go"
QS = "testdata/snapshots/quantitystrings.divergences.tsv"
PLAN = "docs/plans/010-to-quantity.md"
MOVED = "answers moved in quantitystrings"


# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The number.
    ("'.5' read again",
     (NUMBER, '\tif p.whole == "" {\n', '\tif p.whole == "" && (j >= len(s) || s[j] != \'.\') {\n'),
     SNAPSHOTS, "answers moved in decimalstrings"),
    ("the exponent not read",
     (NUMBER, "\tif j < len(s) && (s[j] == 'e' || s[j] == 'E') {", "\tif false && j < len(s) && (s[j] == 'e' || s[j] == 'E') {"),
     SNAPSHOTS, MOVED),
    ("the Quantity brought to the Decimal scale",
     (READ, "return decimal.NewFromBigInt(coefficient, int32(written)), true",
      "return ScaleDecimal(decimal.NewFromBigInt(coefficient, int32(written))), true"), SNAPSHOTS, MOVED),
    ("the range not applied",
     (READ, "if lead >= maxDecimalDigitsBeforePoint || ", "if "), SNAPSHOTS, MOVED),
    ("the precision not applied",
     (READ, " || max(lead+1, 0)+places > maxDecimalPrecision {", " {"), BUDGET,
     "--- FAIL: TestAQuantityStringOfMillionsOfBytesIsReadInBoundedResources"),
    ("a zero's places not bounded",
     (READ, "\t\tif places > maxDecimalPrecision {\n\t\t\treturn decimal.Decimal{}, false", "\t\tif false {\n\t\t\treturn decimal.Decimal{}, false"),
     GIVEN, "want null"),
    # The whitespace.
    ("a line break not trimmed",
     (READ, "\\x08\\t\\n\\x0b\\x0c\\r", "\\x08\\t\\x0b\\x0c"), SNAPSHOTS, MOVED),
    ("a line break between the number and the unit",
     (READ, 'unitSeparator = " \\t"', 'unitSeparator = " \\t\\n"'), SNAPSHOTS, MOVED),
    ("a space required before the unit",
     (READ, "\trest := strings.TrimLeft(s[end:], unitSeparator)\n",
      "\trest := strings.TrimLeft(s[end:], unitSeparator)\n\tif len(rest) == len(s[end:]) && rest != \"\" {\n\t\treturn nil, nil\n\t}\n"),
     SNAPSHOTS, MOVED),
    # The unit.
    ("the default unit lost",
     (READ, 'return fptypes.NewQuantityFromDecimal(value, "1"), nil', 'return fptypes.NewQuantityFromDecimal(value, ""), nil'),
     SNAPSHOTS, MOVED),
    ("an unquoted unit read",
     (READ, "\t// A string literal: a quote, the unit, and the closing quote last.\n",
      "\tif rest[0] != '\\'' {\n\t\treturn fptypes.NewQuantityFromDecimal(value, rest), nil\n\t}\n"),
     SNAPSHOTS, MOVED),
    ("what follows the closing quote ignored",
     (READ, "strings.IndexByte(rest[1:], '\\'') != len(rest)-2 {\n\t\treturn nil, nil\n\t}\n\tunit := rest[1 : len(rest)-1]",
      "strings.IndexByte(rest[1:], '\\'') < 0 {\n\t\treturn nil, nil\n\t}\n\tunit := rest[1 : 1+strings.IndexByte(rest[1:], '\\'')]"),
     SNAPSHOTS, MOVED),
    ("the unit kept as a slice of the string",
     (READ, "\tunit = strings.Clone(unit)\n", ""), run("TestAQuantityDoesNotKeepTheStringItWasReadFrom", package="./funcs/"),
     "from strings of ten million bytes keep"),
    ("the unit not validated",
     (READ, "if service.Validate(unit) != nil {", "if false && service.Validate(unit) != nil {"), SNAPSHOTS, MOVED),
    ("a calendar keyword left out",
     ("types/calendar_sum.go", '"millisecond": true, ', ""), KEYWORDS, "and the grammar has it"),
    ("a keyword read case-insensitively",
     (READ, "if !cqltypes.IsCalendarKeyword(unit) {", "if !cqltypes.IsCalendarKeyword(strings.ToLower(unit)) {"),
     SNAPSHOTS, MOVED),
    # The overloads and the forms.
    ("an Integer without the default unit",
     ("funcs/type_ops.go", 'decimal.NewFromInt(val.Value()), "1")', 'decimal.NewFromInt(val.Value()), "")'), SNAPSHOTS, MOVED),
    ("a Decimal without the default unit",
     ("funcs/type_ops.go", 'NewQuantityFromDecimal(val.Value(), "1")', 'NewQuantityFromDecimal(val.Value(), "")'), SNAPSHOTS, MOVED),
    ("convert fails again",
     (EVAL, 'if strings.EqualFold(nt.Name, "Quantity") &&', 'if false && strings.EqualFold(nt.Name, "Quantity") &&'),
     SNAPSHOTS, MOVED),
    ("a type with no overload converted to null",
     ("funcs/type_ops.go", '\treturn nil, fmt.Errorf("cannot convert %s to Quantity", v.Type())', "\treturn nil, nil"),
     run("TestConvertToQuantityKeepsItsOverloads"),
     "want the error that it cannot convert"),
    ("convert to Quantity in the cast too",
     (EVAL, "\tcase \"quantity\":\n\t\tif q, ok := v.(fptypes.Quantity); ok {\n\t\t\treturn q, nil\n\t\t}\n",
      "\tcase \"quantity\":\n\t\treturn funcs.ToQuantity(v)\n"),
     SNAPSHOTS, "answers moved in"),
    ("ConvertsToQuantity true of a null",
     (EVAL, "funcs.ToQuantity(src)\n\t\treturn fptypes.NewBoolean(convErr == nil && result != nil)",
      "funcs.ToQuantity(src)\n\t\treturn fptypes.NewBoolean(convErr == nil && (result != nil || true))"), FORMS,
     "ConvertsToQuantity("),
    ("convert apart from ToQuantity",
     (EVAL, "\t\t\t\tconverted, err := funcs.ToQuantity(operand)\n",
      "\t\t\t\tconverted, err := funcs.ToQuantity(fptypes.NewString(strings.TrimSpace(operand.String())))\n"),
     FORMS, "to Quantity = "),
    ("whitespace around read apart",
     (READ, "\ts = trimmed(s)\n", "\tif strings.HasPrefix(s, \" \") {\n\t\treturn nil, nil\n\t}\n\ts = trimmed(s)\n"),
     FORMS, "and ToQuantity("),
    # The oracle written apart.
    ("the oracle answers null to everything",
     (ORACLE, "\treturn fptypes.NewQuantityFromDecimal(value, unit), true\n}", "\treturn fptypes.Quantity{}, false\n}"),
     GIVEN, "want null"),
    ("the oracle reads no unit",
     (ORACLE, '\tunit := "1"\n\tif m[5] != "" {', '\tunit := "1"\n\tif false {'), GIVEN, "QuantityFromString("),
    # The area.
    ("a deliberate divergence dropped from its file",
     (QS, "\nD6\tjava\tdeliberate\t", "\nD6-gone\tjava\tdeliberate\t"), REFERENCES, "are not the ones recorded"),
    ("a deliberate divergence the plan does not name",
     (PLAN, "D336, D337, D338", "D336, D338"), CONSISTENT, "does not name D337"),
    # D851 became deliberate in plan 011; the oracle is held on D39 of decimalliterals.
    ("an own defect expected as this engine answers",
     ("testdata/snapshots/decimalliterals.divergences.tsv", '\nD39\tjava\town defect\t"1.0"\t',
      '\nD39\tjava\town defect\t"000000000000000000000000000001.0"\t'),
     CONSISTENT, "an own defect expects an answer other than this engine's"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"quantitystrings", 894, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "quantitystrings.cql is not in snapshotAreas"),
    ("the area with another number of definitions",
     ("engine_snapshot_test.go", '{"quantitystrings", 894, true},', '{"quantitystrings", 893, true},'), SNAPSHOTS,
     "the manifest says 893"),
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
