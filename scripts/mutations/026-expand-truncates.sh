#!/bin/sh
# Mutations of what plan 026 holds (docs/plans/026-expand-truncates-to-per.md):
# expand … per reads each boundary with start of / end of at its own precision,
# truncates it to per's precision (a number toward zero, a temporal in its own
# digits and offset), drops what is less precise than per, merges the list once
# truncated, counts whole partitions from the first interval, and infers per
# over the whole list; and the area `expansions` holds it to its oracle.
# Each is applied to a copy of the repository; the command it names has to fail,
# and its output has to say why.
# Usage: sh scripts/mutations/026-expand-truncates.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-026.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, re, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-failfast", "-run", "^(" + "|".join(names) + ")$", package]


X = "funcs/interval_expand.go"
T = "engine_expand_per_test.go"
def t(name):
    return run(name)
EXAMPLES = t("TestTheExamplesOfExpand")
TRUNC = t("TestABoundMorePreciseThanPerIsTruncated")
DIGITS = t("TestATruncatedDateTimeKeepsItsOwnDigitsAndOffset")
WHOLE = t("TestOnlyWholeIntervalsOfPerAreContributed")
INFER = t("TestPerIsInferredFromTheWholeList")
COARSE = t("TestABoundLessPreciseThanPerContributesNothing")
OPEN = t("TestAnOpenBoundIsItsSuccessorThenTruncated")
CALENDAR = t("TestTheCalendarIsWalkedFromTheStart")
ORDER = t("TestTheInputIsCollapsedOrderedAndAnchoredAtTheFirst")
ORACLE = t("TestTheExpansionsAreTheirOracle")
AREA = ["go", "test", "-count=1", "-failfast", "-run", "^TestAnswersMatchTheSnapshots$/^expansions$", "."]
MOVED = "moved in expansions"

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # Truncation.
    ("a Decimal is not truncated",
     (X, "\tif d.Exponent() >= -scale {\n\t\treturn d\n\t}", "\tif true {\n\t\treturn d\n\t}"), TRUNC, "want"),
    ("a Decimal is not truncated: the area",
     (X, "\tif d.Exponent() >= -scale {\n\t\treturn d\n\t}", "\tif true {\n\t\treturn d\n\t}"), AREA, MOVED),
    ("a Decimal is rounded instead of truncated",
     (X, "\treturn d.Truncate(scale)\n}", "\treturn d.Round(scale)\n}"), AREA, MOVED),
    ("a DateTime is not cut to the day",
     (X, "\t\tif rank >= 4 {\n\t\t\ttext += fmt.Sprintf(\"T%02d\", t.Hour())", "\t\tif rank >= 3 {\n\t\t\ttext += fmt.Sprintf(\"T%02d\", t.Hour())"),
     AREA, MOVED),
    ("a truncated DateTime loses its offset",
     (X, "writes := t.HasTZ() && rank >= 4", "writes := false && t.HasTZ() && rank >= 4"), DIGITS, "want"),
    ("an Integer does not gain the precision of per",
     (X, "\t\t\tr.hi = r.hi.Add(one).Sub(r.ulp)\n", "\n"), EXAMPLES, "want"),
    # Whole partitions and the anchor.
    ("the last partition may end past the high bound",
     (X, "for off <= span && span-off >= step-1 {", "for off <= span {"), WHOLE, "want"),
    ("the last temporal partition may end past the high bound",
     (X, "for ; start+stepMs-unitMs <= hi; start += stepMs {", "for ; start <= hi; start += stepMs {"), AREA, MOVED),
    ("each Decimal interval is counted from its own start",
     (X, "\t\tif !seen {\n\t\t\tanchor = r.lo", "\t\tif _ = seen; true {\n\t\t\tanchor = r.lo"), AREA, MOVED),
    ("each Integer interval is counted from its own start",
     (X, "\t\toff := uint64(r.lo) - anchor //nolint:gosec // distance, exact: anchor <= lo\n",
      "\t\tanchor = uint64(r.lo) //nolint:gosec // mutation\n\t\toff := uint64(r.lo) - anchor //nolint:gosec // distance, exact: anchor <= lo\n"),
     ORDER, "want"),
    ("each temporal interval is counted from its own start",
     (X, "\tfor _, r := range ranges {\n\t\t// The first partition that starts on or after the low bound: partition",
      "\tfor _, r := range ranges {\n\t\tanchor = r.lo\n\t\t// The first partition that starts on or after the low bound: partition"),
     AREA, MOVED),
    # Merging, after the cut.
    ("temporal intervals are not merged",
     (X, "if c, err := compareVals(r.lo, oneUnit(last.hi)); err == nil && c <= 0 {", "if c, err := compareVals(r.lo, oneUnit(last.hi)); err == nil && c < -99 {"),
     ORDER, "want"),
    ("temporal intervals that only meet are not merged",
     (X, "if c, err := compareVals(r.lo, oneUnit(last.hi)); err == nil && c <= 0 {", "if c, err := compareVals(r.lo, last.hi); err == nil && c <= 0 && oneUnit != nil {"),
     ORDER, "want"),
    ("Integer intervals that only meet are not merged",
     (X, "r.lo <= last.hi+1 {", "r.lo <= last.hi {"), AREA, MOVED),
    ("Decimal intervals that only meet are not merged",
     (X, "!r.lo.GreaterThan(merged[n-1].hi.Add(r.ulp))", "!r.lo.GreaterThan(merged[n-1].hi)"), AREA, MOVED),
    # What contributes nothing, and per inferred.
    ("a bound less precise than per contributes",
     (X, "if lr < perRank || hr < perRank {", "if lr < perRank-99 || hr < perRank-99 {"), COARSE, "want"),
    ("per is inferred from the finest bound",
     (X, "\t\t\t\t\trank = min(rank, r)", "\t\t\t\t\trank = r"), INFER, "want"),
    ("a week is a day",
     (X, "amount, unit = amount*7, \"day\"", "amount, unit = amount, \"day\""), AREA, MOVED),
    # Open bounds.
    ("an open temporal low bound is read as closed",
     (X, "\t\tif !iv.LowClosed {\n\t\t\tif low, err = cqltypes.Successor(low)", "\t\tif false {\n\t\t\tif low, err = cqltypes.Successor(low)"),
     OPEN, "want"),
    ("an open Decimal low bound is read as closed",
     (X, "perGiven, \"\")\n\t\t\tif !iv.LowClosed {\n\t\t\t\tr.lo = r.lo.Add(decimalEpsilon)", "perGiven, \"\")\n\t\t\tif !iv.LowClosed {\n\t\t\t\tr.lo = r.lo.Add(decimal.Zero)"), AREA, MOVED),
    # The oracle and its files.
    ("the oracle gives a row another answer",
     ("testdata/expansions/oracle.tsv", "\nD1\t", "\nD1\tList[Integer 0], "), ORACLE, "the oracle gives"),
    ("the snapshot gives a row another answer",
     ("testdata/snapshots/expansions.answers.tsv", "\nD2\t", "\nD2\tnull"), ORACLE, "the oracle gives"),
    ("a row of the oracle is not in the area",
     ("testdata/expansions/oracle.tsv", "\nD1\t", "\nD99999\tList[]\nD1\t"), ORACLE, "is in the oracle and not in the area"),
]

# A deliberate verdict of the area names its row in plan 026 (TestTheDivergencesAreConsistent).
plan = open(os.path.join(repo, "docs/plans/026-expand-truncates-to-per.md")).read()
divergences = [l.split("\t") for l in open(os.path.join(repo, "testdata/snapshots/expansions.divergences.tsv")).read().split("\n")
               if l and not l.startswith("#")]
named_once = [d[0] for d in divergences if d[2] == "deliberate" and "docs/plans/026-expand-truncates-to-per.md" in d[4]
              and len(re.findall(r"(?<![\w-])" + d[0] + r"(?![\w-])", plan)) == 1 and f" {d[0]} " in plan]
if not named_once:
    print("NO DELIBERATE ROW of plan 026 to mutate, named once in the plan as ` Dn `")
    sys.exit(2)
mutations.append(
    ("a deliberate row that cites a plan that does not name it",
     ("docs/plans/026-expand-truncates-to-per.md", f" {named_once[0]} ", " DX "), run("TestTheDivergencesAreConsistent"),
     "which does not name " + named_once[0]))

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
    source = "\n" + open(target).read()
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
