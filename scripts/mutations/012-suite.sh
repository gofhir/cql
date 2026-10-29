#!/bin/sh
# Mutations of what plan 012 holds for the conformance suite at cql-tests
# f724ee4 (docs/plans/012-suite-cql-tests.md): the vendored files and their
# manifest, the list of known failures and what each records, the area that
# holds the references' answers to them, and the version range of a test. Each
# is applied to a copy of the repository; the command it names has to fail,
# and its output has to say why.
# Usage: sh scripts/mutations/012-suite.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-012.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()


def run(*names, package="."):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", package]


CONFORMANCE = run("TestConformance", package="./conformance/")
TIE = run("TestEveryKnownFailureIsOfTheSuite", package="./conformance/")
VENDORED = run("TestTheConformanceSuiteIsTheVendoredOne", package="./conformance/")
RANGE = run("TestTheVersionsOfATestAreARange", package="./conformance/")
SNAPSHOTS = run("TestAnswersMatchTheSnapshots")
KNOWN = "conformance/known_failures_test.go"
RUNNER = "conformance/conformance_test.go"
LIST = "conformance/testdata/CqlListOperatorsTest.xml"
GAPS = "testdata/snapshots/suitegaps.cql"
SKIP = '\t"SkipNegative":                      {group: "G3d", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`},\n'

# name, (path, old, new), command, text the failure has to hold
mutations = [
    ("a known failure left out of the list",
     (KNOWN, SKIP, ""), CONFORMANCE, "SkipNegative"),
    ("a test that passes put in the list",
     (KNOWN, SKIP, SKIP + '\t"SliceAll": {group: "G1", errs: false, suite: ""},\n'), CONFORMANCE,
     "SliceAll passes now"),
    ("a name the suite does not have",
     (KNOWN, SKIP, SKIP + '\t"NoSuchTest": {group: "G3d", errs: false, suite: ""},\n'), TIE,
     "the suite does not have"),
    ("a group failureGroups does not have",
     (KNOWN, 'group: "G3d", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`',
      'group: "G9", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`'), TIE, "is not in failureGroups"),
    ("a test of CQL 1.0 in G1",
     (KNOWN, 'group: "G3d", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`',
      'group: "G1", errs: false, suite: `1.0.. invalid="" outputs=["{ }"]`'), TIE, "G1 is for the tests of a CQL after"),
    ("a test whose range ended before 1.5.3 in G1",
     (KNOWN, SKIP, SKIP + '\t"DateTimeComponentFromTimezoneOffset": {group: "G1", errs: false, suite: ""},\n'), TIE,
     "G1 is for the tests of a CQL after"),
    ("a known failure with errs that passes",
     (KNOWN, SKIP, SKIP + '\t"SliceAll": {group: "G1", errs: true, suite: ""},\n'), CONFORMANCE,
     "SliceAll passes now"),
    ("a known failure that fails another way",
     (KNOWN, 'group: "G3d", errs: false', 'group: "G3d", errs: true'), CONFORMANCE, "fails another way now"),
    ("the suite changes what it asks of a known failure",
     (LIST, "\t\t\t<expression>Skip({ 1, 2, 3 }, -1)</expression>\n\t\t\t<output>{ }</output>",
      "\t\t\t<expression>Skip({ 1, 2, 3 }, -1)</expression>\n\t\t\t<output>{ 1 }</output>"), TIE, "classify it again"),
    ("the area with another expression than the suite's",
     (GAPS, "Skip({ 1, 2, 3 }, -1) // SkipNegative", "Skip({ 1, 2, 3 }, -2) // SkipNegative"), TIE, "and the suite has"),
    ("a known failure without its definition in the area",
     (GAPS, "define D44: Skip({ 1, 2, 3 }, -1) // SkipNegative\n", ""), TIE,
     "has no definition in suitegaps.cql"),
    ("the answer to a known failure moves, and it still fails",
     ("funcs/list_advanced.go", "\tif n <= 0 {\n\t\treturn c\n\t}\n\tif n >= c.Count()",
      "\tif n < 0 {\n\t\treturn c[:1]\n\t}\n\tif n == 0 {\n\t\treturn c\n\t}\n\tif n >= c.Count()"), SNAPSHOTS,
     "answers moved in suitegaps"),
    ("a byte of a vendored file changed",
     (LIST, "<expression>Skip({ 1, 2, 3 }, -1)</expression>", "<expression>Skip({ 1, 2, 3 }, -2)</expression>"),
     VENDORED, "CqlListOperatorsTest.xml: git blob"),
    ("a SHA-256 of the manifest corrupted",
     ("conformance/testdata/SOURCE.tsv", "\t7f328e6df4833409cb1c7eed108fcdf35f82bf1223b384a57b0d7b95677b0cd5",
      "\t0f328e6df4833409cb1c7eed108fcdf35f82bf1223b384a57b0d7b95677b0cd5"), VENDORED, "CqlTypesTest.xml: SHA-256"),
    ("the Floor contradiction kept after upstream settled it",
     (RUNNER, "var upstreamContradictions = map[string]string{\n",
      'var upstreamContradictions = map[string]string{\n\t"FloorIntegerGreaterThanMaxInteger": "kept",\n'),
     CONFORMANCE, "passes now, so upstream has settled it"),
    ("the area left out of what the references run",
     ("scripts/references.sh", " decimalliterals suitegaps ", " decimalliterals "),
     run("TestEveryAreaIsRunByTheReferences"), "scripts/references.sh does not run it"),
    ("versionTo not read",
     ("conformance/xmlmodel.go", 'xml:"versionTo,attr"', 'xml:"versionto,attr"'), RANGE,
     "DateTimeComponentFromTimezoneOffset, of CQL 1.4..: applies"),
    ("the end of a range left out",
     (KNOWN, '(to == "" || compareVersions(v, to) <= 0)', "true"), RANGE, "DateTimeComponentFromTimezoneOffset"),
    ("a deliberate divergence the plan does not name",
     ("docs/plans/012-suite-cql-tests.md", "D46, D47 y D48", "D46 y D47"), run("TestTheDivergencesAreConsistent"),
     "does not name D48"),
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
