#!/bin/sh
# Mutations of what plan 019 holds (docs/plans/019-date-promotion.md): a Date next
# to a DateTime is converted to a DateTime, sema records the conversion against
# the Date side with no function (DatePromotion), and the evaluator applies it
# with ToDateTime and the evaluation's offset. Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# The data mutations edit the answers, the oracle and the promotions of the area
# `mixeddates`.
# Not here, until the divergences of the area are classified (they are written
# unclassified): a deliberate row that cites a plan that does not name it, and a
# divergence left unclassified. They are tried, against the first classified row
# of mixeddates.divergences.tsv, only when there is one.
# Usage: sh scripts/mutations/019-date-promotion.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-019.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, re, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-failfast", "-run", "^(" + "|".join(names) + ")$", package]


SEMA_PROMOTED = run("TestADateNextToADateTimeIsRecordedAsPromoted", package="./sema")
SEMA_NOT = run("TestADateIsNotPromotedWhereTheTranslatorDoesNotConvert", package="./sema")
SEMA_FUNCTIONS = run("TestAFunctionArgumentAndAParameterDefaultAreConverted", package="./sema")
SEMA_MODEL = run("TestAModelFunctionIsFollowedByTheDatePromotion", package="./sema")
HAND = run("TestADateNextToADateTimeIsADateTime")
HAND_NULL = run("TestACoalesceWithANullFirstAndAUnionOfListsAreNotConverted")
HAND_PARAM = run("TestTheDefaultOfATypedParameterIsConverted")
HAND_FHIR = run("TestAFHIRDateNextToADateTimeIsConverted")
HAND_INTERVAL = run("TestAnIntervalPromotedToDateTimeHasDateTimeBounds")
PROMOTIONS = run("TestTheMixedDatesPromotionsAreTheTranslators")
ORACLE = run("TestTheMixedDatesAreTheirOracle")
PAIRS = run("TestAnImplicitConversionAnswersAsTheExplicitOne")
EVAL = "eval/evaluator.go"
CHECK = "sema/check.go"
CONVERT = "sema/convert.go"
INFER = "sema/infer.go"
OPERATORS = "sema/operators.go"
QUERY = "sema/query.go"


def read(path):
    return open(os.path.join(repo, path)).read()


# A boolean row of the oracle, and a pair of the area, for the data mutations.
oracle = [l.split("\t") for l in read("testdata/mixeddates/oracle.tsv").split("\n") if "\t" in l]
flip = next((n, a) for n, a in oracle if a in ("Boolean true", "Boolean false"))
flipped = "Boolean false" if flip[1] == "Boolean true" else "Boolean true"
implicit = next(l.split("\t")[0] for l in read("testdata/mixeddates/pairs.tsv").split("\n") if "\t" in l)
promos = dict(l.split("\t") for l in read("testdata/snapshots/mixeddates.promotions.tsv").split("\n") if "\t" in l)
promoted = next(n for n, c in promos.items() if c not in ("rejected", "0"))
unpromoted = next(n for n, c in promos.items() if c == "0")
answers = dict(l.split("\t", 1) for l in read("testdata/snapshots/mixeddates.answers.tsv").split("\n") if "\t" in l)

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("DatePromotion is always false",
     (CONVERT, "func DatePromotion(from, to Type) bool {\n\tif from == nil || to == nil {",
      "func DatePromotion(from, to Type) bool {\n\tif true {\n\t\treturn false\n\t}\n\tif from == nil || to == nil {"), SEMA_PROMOTED,
     "promotions recorded on [], want [System.Date]"),
    ("DatePromotion is always true",
     (CONVERT, "\t\treturn Equal(t, DateTime) && Equal(from, Date)\n", "\t\treturn true\n"), PROMOTIONS,
     "the translator wrote 0 ToDateTime nodes, sema records"),
    ("convert drops the Date promotion",
     (CHECK, " && !DatePromotion(c.types[expr], conv.Target) {\n\t\treturn", " {\n\t\treturn"), SEMA_PROMOTED,
     "promotions recorded on [], want [System.Date]"),
    ("Coalesce converts no Date",
     (QUERY, "c.promoteRecorded(arg, t, dates)", "c.promoteRecorded(arg, t, dates && false)"), SEMA_PROMOTED,
     "Coalesce(@2020-03-01, @2020-03-01T10:00:00.000): promotions recorded on [], want [System.Date]"),
    ("Coalesce with a null first converts",
     (QUERY, "dates := !isAny(c.types[exprs[0]])", "dates := true"), SEMA_NOT,
     "Coalesce(null, @2020-03-01, @2020-03-01T10:00:00.000): promotions recorded on [System.Date], want none"),
    ("an interval constructor converts no Date",
     (INFER, "c.promoteRecorded(e.Low, point, true)", "c.promoteRecorded(e.Low, point, false)"), SEMA_PROMOTED,
     "Interval[@2020-03-01, @2020-03-01T10:00:00.000]: promotions recorded on [], want [System.Date]"),
    ("a comparison is not recorded",
     (OPERATORS, "\tc.promoteDatePair(e.Left, left, e.Right, right)\n\tif _, ok := Convertible(left, right, c.model); ok {",
      "\tif _, ok := Convertible(left, right, c.model); ok {"), SEMA_PROMOTED,
     "@2020-03-01 = @2020-03-01T10:00:00.000: promotions recorded on [], want [System.Date]"),
    ("a between operand is not recorded",
     (INFER, "\tc.promoteDate(e.Operand, operand, want)\n\treturn Boolean", "\treturn Boolean"), SEMA_PROMOTED,
     "@2020-03-01 between @2020-03-01T10:00:00.000 and @2020-03-05T10:00:00.000: promotions recorded on [], want [System.Date]"),
    ("a time phrase is not recorded",
     (INFER, "\tc.promoteTiming(e, left, right)\n\treturn Boolean", "\treturn Boolean"), SEMA_PROMOTED,
     "@2020-03-01 same day as @2020-03-01T10:00:00.000: promotions recorded on [], want [System.Date]"),
    ("a membership with a precision is not recorded",
     (INFER, "\tc.promoteDateIn(element, left, container, right)\n", "\t_ = container\n"), SEMA_PROMOTED,
     "promotions recorded on [], want"),
    ("a union of intervals is not recorded",
     (OPERATORS, "\t\t\tc.promoteDatePair(e.Left, left, e.Right, right)\n\t\t\treturn &Interval{Point:", "\t\t\treturn &Interval{Point:"), SEMA_PROMOTED,
     "union Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]: promotions recorded on [], want [Interval<System.Date>]"),
    ("a union of lists is converted",
     (OPERATORS, "\t\t\treturn &List{Element: Common(l.Element, r.Element, c.model)}", "\t\t\tc.promoteDatePair(e.Left, left, e.Right, right)\n\t\t\treturn &List{Element: Common(l.Element, r.Element, c.model)}"), SEMA_NOT,
     "{ @2020-03-01 } union { @2020-03-01T10:00:00.000 }: promotions recorded on [List<System.Date>], want none"),
    ("a refused signature is converted: an interval during a point",
     (INFER, "\tcase ast.TimingIncludedIn, ast.TimingDuring:\n\t\tif c.promoteDateInDirect(e.Left, left, e.Right, right) {\n\t\t\treturn\n\t\t}",
      "\tcase ast.TimingIncludedIn, ast.TimingDuring:\n\t\tif c.promoteDateInDirect(e.Left, left, e.Right, right) || c.promoteDateInDirect(e.Right, right, e.Left, left) {\n\t\t\treturn\n\t\t}"), SEMA_NOT,
     "during @2020-03-01: promotions recorded on [System.Date], want none"),
    ("a difference between is not recorded",
     (INFER, "\t\tc.promoteDatePair(e.Low, c.expectTemporalOrNumeric(e.Low), e.High, c.expectTemporalOrNumeric(e.High))\n\t\treturn c.record(e, Integer)\n\tcase *ast.DifferenceBetween:",
      "\t\tc.expectTemporalOrNumeric(e.Low)\n\t\tc.expectTemporalOrNumeric(e.High)\n\t\treturn c.record(e, Integer)\n\tcase *ast.DifferenceBetween:"), SEMA_PROMOTED,
     "days between @2020-03-01 and @2020-03-01T10:00:00.000: promotions recorded on [], want [System.Date]"),
    ("a function argument is not converted",
     (QUERY, "\t\t\t\tif i < len(exprs) {\n\t\t\t\t\tc.convert(exprs[i], conv)", "\t\t\t\tif i < len(exprs) && conv.Function != \"\" {\n\t\t\t\t\tc.convert(exprs[i], conv)"), SEMA_FUNCTIONS,
     "promotions recorded on [], want [System.Date]"),
    ("the default of a typed parameter is not converted",
     (CHECK, "\t\t\tc.promoteDate(p.Default, c.infer(p.Default), c.params[p.Name])", "\t\t\tc.infer(p.Default)"), SEMA_FUNCTIONS,
     "promotions recorded on [], want"),
    ("a model function and the promotion are not one conversion",
     (CHECK, "\t\tif prev.Function != \"\" && DatePromotion(prev.Target, conv.Target) {", "\t\tif false && prev.Function != \"\" && DatePromotion(prev.Target, conv.Target) {"), SEMA_MODEL,
     "birthDate converts by"),
    ("the FHIR path does not convert",
     (EVAL, "\treturn e.promoteDates(converted, conv.Target)\n}", "\treturn converted, nil\n}"), HAND_FHIR,
     "is DateTime = false, want true"),
    ("the evaluator converts no Date directly",
     (EVAL, "promoted, err := e.promoteDates(promoteNumbers(v, conv.Target), conv.Target)\n\t\treturn promoted, err", "return promoteNumbers(v, conv.Target), nil"), HAND,
     "is DateTime = false, want true"),
    ("the evaluator converts no Date inside a list",
     (EVAL, "promoted, changed, err := e.promoteDatesIn(el, t.Element)", "promoted, changed, err := e.promoteDatesIn(el, sema.Integer)"), HAND,
     "is DateTime = false, want true"),
    ("the evaluator converts no low bound of an interval",
     (EVAL, "low, lowChanged, err := e.promoteDatesIn(iv.Low, t.Point)", "low, lowChanged, err := e.promoteDatesIn(iv.Low, sema.Integer)"), HAND_INTERVAL,
     "want Interval[21, 44]"),
    ("the evaluator converts no high bound of an interval",
     (EVAL, "high, highChanged, err := e.promoteDatesIn(iv.High, t.Point)", "high, highChanged, err := e.promoteDatesIn(iv.High, sema.Integer)"), HAND,
     "is DateTime = false, want true"),
    ("a typed parameter is converted by sema and not applied",
     (EVAL, "\tdate, ok := v.(fptypes.Date)\n\tif !ok {", "\tdate, ok := v.(fptypes.Date)\n\tif !ok || true {"), HAND_PARAM,
     "is DateTime = false, want true"),
    ("the area left out of the manifest",
     ("engine_snapshot_test.go", '\t{"mixeddates", 1385, true},\n', ""), run("TestEveryCorpusIsInTheManifest"),
     "mixeddates.cql is not in snapshotAreas"),
    ("the area left out of the references",
     ("scripts/references.sh", " intervalbounds mixeddates\"", " intervalbounds\""), run("TestEveryAreaIsRunByTheReferences"),
     "mixeddates is in snapshotAreas and scripts/references.sh does not run it"),
    ("an answer of the area moved away from the oracle",
     ("testdata/snapshots/mixeddates.answers.tsv", f"\n{flip[0]}\t{flip[1]}\n", f"\n{flip[0]}\t{flipped}\n"), ORACLE,
     f"the oracle gives {flip[1]}"),
    ("an answer of the oracle moved away from the engine",
     ("testdata/mixeddates/oracle.tsv", f"\n{flip[0]}\t{flip[1]}\n", f"\n{flip[0]}\t{flipped}\n"), ORACLE,
     f"the oracle gives {flipped}"),
    ("the promotions check accepts a missing ToDateTime: the translator's node removed",
     ("testdata/snapshots/mixeddates.promotions.tsv", f"\n{promoted}\t{promos[promoted]}\n", f"\n{promoted}\t0\n"), PROMOTIONS,
     "the translator wrote 0 ToDateTime nodes, sema records"),
    ("the promotions check accepts a missing DatePromotion: a translator's node added",
     ("testdata/snapshots/mixeddates.promotions.tsv", f"\n{unpromoted}\t0\n", f"\n{unpromoted}\t2\n"), PROMOTIONS,
     "the translator wrote 2 ToDateTime nodes, sema records 0"),
    ("an implicit form moved away from its twin",
     ("testdata/snapshots/mixeddates.answers.tsv", f"\n{implicit}\t{answers[implicit]}\n", f"\n{implicit}\tBoolean maybe\n"), PAIRS,
     f"{implicit} answers Boolean maybe and its twin"),
]

# Not until the divergences of the area are classified.
divergences = [l.split("\t") for l in read("testdata/snapshots/mixeddates.divergences.tsv").split("\n")
               if l and not l.startswith("#")]
unclassified = {"unclassified", "undecided"}
settled = next((d for d in divergences if d[2] not in unclassified), None)
if settled is None:
    print("pending      a deliberate row that cites a plan that does not name it, and a divergence left unclassified: "
          "the divergences of mixeddates are not classified yet")
else:
    plan = read("docs/plans/019-date-promotion.md")
    named_once = [d[0] for d in divergences if d[2] == "deliberate"
                  and len(re.findall(r"(?<![\w-])" + d[0] + r"(?![\w-])", plan)) == 1]
    if named_once:
        mutations.append(
            ("a deliberate row that cites a plan that does not name it",
             ("docs/plans/019-date-promotion.md", f" {named_once[0]},", " DX,"), run("TestTheDivergencesAreConsistent"),
             "which does not name " + named_once[0]))
    line = "\t".join(settled)
    mutations.append(
        ("a classified divergence of the area left unclassified",
         ("testdata/snapshots/mixeddates.divergences.tsv", f"\n{settled[0]}\t{settled[1]}\t{settled[2]}\t",
          f"\n{settled[0]}\t{settled[1]}\tunclassified\t"), run("TestThePendingVerdictsDoNotGrow"),
         "has no verdict and is not in pending-verdicts.tsv"))

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
    source = "\n" + open(target).read()  # a line at the top of a file is still preceded by one
    if source.count(old) != 1:
        print(f"NOT APPLIED  {name}: {path}: found {source.count(old)} times")
        survived += 1
        continue
    open(target, "w").write(source.replace(old, new)[1:])
    r = subprocess.run(command, cwd=copy, capture_output=True, text=True, env=CLEAN_ENV)
    output = r.stdout + r.stderr
    if "build failed" in output or "[setup failed]" in output:
        print(f"BUILD FAILED {name}: a mutation that does not compile proves nothing\n{output[-1500:]}")
        survived += 1
    elif r.returncode == 0 or text not in output:
        print(f"SURVIVED     {name}: expected a failure saying {text!r}\n{output[-1500:]}")
        survived += 1
    else:
        print(f"caught       {name}")
sys.exit(1 if survived else 0)
PY
