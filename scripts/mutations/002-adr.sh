#!/bin/sh
# Mutations of the ADR governance and the evidence provenance of increment 002
# (docs/plans/002-adr-semantica-cql.md). Each is applied to a copy of the
# repository; the command it names has to fail, and its output has to say why.
# Usage: sh scripts/mutations/002-adr.sh [work-directory]
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
import hashlib, os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()

ADR = "docs/adr/001-semantica-cql-desde-su-especificacion.md"
CI = ".github/workflows/ci.yml"
ROOT = ["go", "test", "-count=1", "-run", "TestADRsHaveVerification", "."]
CONF = ["go", "test", "-count=1", "-run", "TestTheConformanceSuiteIsTheVendoredOne", "./conformance/"]
ACCEPTED = (ADR, "**Estado:** Propuesto.", "**Estado:** Aceptado.")
NO_TARGETS = [(ADR, "- `make conformance-provenance`: `SOURCE.tsv`", "- conformance-provenance: `SOURCE.tsv`"),
              (ADR, "sale de `make evidence`", "sale de make evidence")]
APPEND = "append"   # old: append new to the file
WRITE = "write"     # old: write the file with new
DELETE = "delete"   # old: remove the file


def regenerated_manifest(copy):
    # Edit a file, then rewrite its manifest row to match: the unit test stays
    # green, and only the comparison with upstream can tell.
    name = "CqlAggregateTest.xml"
    path = os.path.join(copy, "conformance", "testdata", name)
    content = open(path, "rb").read() + b"\n"
    open(path, "wb").write(content)
    blob = hashlib.sha1(b"blob %d\0" % len(content) + content).hexdigest()
    sha = hashlib.sha256(content).hexdigest()
    manifest = os.path.join(copy, "conformance", "testdata", "SOURCE.tsv")
    rows = [f"{name}\t{blob}\t{sha}" if l.startswith(name + "\t") else l
            for l in open(manifest).read().splitlines()]
    open(manifest, "w").write("\n".join(rows) + "\n")


GIT = ["git", "-c", "user.email=m@m", "-c", "user.name=m"]


def commit_all(copy, message):
    subprocess.run(GIT + ["add", "-A"], cwd=copy, check=True)
    subprocess.run(GIT + ["commit", "-qm", message], cwd=copy, check=True)


def initial_debt():
    # The debt as it was introduced, from the repository's history: the one
    # content the script accepts where the base does not have the file. The
    # file shrinks as divergences get verdicts, so its content now is not it.
    path = "testdata/snapshots/pending-verdicts.tsv"
    first = subprocess.run(["git", "-C", repo, "log", "--diff-filter=A", "--format=%H", "--", path],
                           capture_output=True, text=True, check=True).stdout.split()[-1]
    return subprocess.run(["git", "-C", repo, "show", f"{first}:{path}"],
                          capture_output=True, text=True, check=True).stdout


def debt_without_a_base_control(copy):
    debt = os.path.join(copy, "testdata", "snapshots", "pending-verdicts.tsv")
    content = initial_debt()
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    os.remove(debt)
    commit_all(copy, "base")
    open(debt, "w").write(content)
    commit_all(copy, "change")


def debt_without_a_base(copy):
    # The base has no debt file, and the change adds one that is not the
    # recorded debt: the bootstrap is accepted for that content only.
    debt = os.path.join(copy, "testdata", "snapshots", "pending-verdicts.tsv")
    content = initial_debt()
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    os.remove(debt)
    commit_all(copy, "base")
    open(debt, "w").write(content + "integer\tD0\tjava\n")
    commit_all(copy, "change")


def push_of_two(copy, debt_line):
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    commit_all(copy, "base")
    before = subprocess.run(["git", "rev-parse", "HEAD"], cwd=copy, capture_output=True, text=True, check=True).stdout.strip()
    with open(os.path.join(copy, "testdata", "snapshots", "pending-verdicts.tsv"), "a") as debt:
        debt.write(debt_line)
    with open(os.path.join(copy, "CHANGELOG.md"), "a") as changelog:
        changelog.write("\n")
    commit_all(copy, "first")
    with open(os.path.join(copy, "README.md"), "a") as readme:
        readme.write("\n")
    commit_all(copy, "unrelated")
    open(os.path.join(copy, "event.json"), "w").write('{"before": "%s"}' % before)


def debt_in_a_push_of_two(copy):
    # A push whose first commit adds debt and whose second changes something
    # else: compared with its last commit alone, the debt did not grow.
    push_of_two(copy, "integer\tD0\tjava\n")


def against_a_base(copy, base_name):
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    commit_all(copy, "base")
    open(os.path.join(copy, "base.txt"), "w").write(base_name)


def unrelated_change_against_the_base(copy):
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    commit_all(copy, "base")
    with open(os.path.join(copy, "README.md"), "a") as readme:
        readme.write("\n")
    commit_all(copy, "change")


def add_to_debt_against_the_base(copy):
    # The base is a commit of the copy as it is; the change unclassifies a
    # divergence and adds it to the debt, which the test of the repository
    # accepts, since the two agree.
    git = ["git", "-c", "user.email=m@m", "-c", "user.name=m"]
    subprocess.run(["git", "init", "-q"], cwd=copy, check=True)
    subprocess.run(git + ["add", "-A"], cwd=copy, check=True)
    subprocess.run(git + ["commit", "-qm", "base"], cwd=copy, check=True)
    manifest = os.path.join(copy, "testdata", "snapshots", "integer.divergences.tsv")
    text = open(manifest).read()
    open(manifest, "w").write(text.replace("D0\tjava\treference defect\t", "D0\tjava\tunclassified\t", 1))
    with open(os.path.join(copy, "testdata", "snapshots", "pending-verdicts.tsv"), "a") as debt:
        debt.write("integer\tD0\tjava\n")
    subprocess.run(git + ["commit", "-qam", "change"], cwd=copy, check=True)


# The mutations that need the network, which make conformance-provenance runs
# (CQL_MUTATIONS_NETWORK=1) so that make check stays offline.
network_mutations = [
    ("a file edited and its manifest rewritten to match",
     regenerated_manifest, ["sh", "scripts/conformance-provenance.sh", "check"], "CqlAggregateTest.xml: recorded"),
]

# (name, edits, command, text its output has to hold)
mutations = [
    ("an ADR names a test that does not exist",
     [(ADR, "`TestADRsHaveVerification`: este ADR", "`TestGhostThatDoesNotExist`: este ADR")],
     ROOT, "TestGhostThatDoesNotExist, which no package"),
    ("an ADR without its verification section",
     [(ADR, "## Verificación\n", "## Comprobación\n")], ROOT, "section"),
    ("an ADR without its state",
     [(ADR, "**Estado:** Propuesto.", "**Situación:** Propuesto.")], ROOT, "no **Estado:**"),
    ("an ADR names a test under testdata, which go test ./... never runs",
     [("testdata/ghost/ghost_test.go", WRITE, "package ghost\n\nimport \"testing\"\n\nfunc TestGhost(t *testing.T) {}\n"),
      (ADR, "`TestADRsHaveVerification`: este ADR", "`TestGhost`: este ADR")],
     ROOT, "TestGhost, which no package"),
    ("a named test is skipped",
     [("conformance/conformance_test.go", "func TestConformance(t *testing.T) {\n",
       "func TestConformance(t *testing.T) {\n\tt.Skip(\"mutation\")\n")],
     ROOT, "TestConformance, which can be skipped"),
    ("an accepted ADR names a make target the verification job does not run",
     [ACCEPTED, (ADR, "- `make conformance-provenance`: `SOURCE.tsv`", "- `make check`: `SOURCE.tsv`")],
     ROOT, "make check, which the ADR verification job does not run"),
    ("an accepted ADR names a benchmark, which make evidence does not run",
     [ACCEPTED, (ADR, "`TestADRsHaveVerification`: este ADR", "`BenchmarkEvaluateFromCold`: este ADR")],
     ROOT, "BenchmarkEvaluateFromCold, which make evidence does not run"),
    ("an accepted ADR without the verification job",
     [ACCEPTED, (CI, "  adr-verification:\n", "  adr-checks:\n")], ROOT, "CI does not have the ADR verification job"),
    ("an accepted ADR whose verification job is off (if: false)",
     [ACCEPTED, (CI, "    name: ADR verification\n", "    name: ADR verification\n    if: false\n")], ROOT, "CI does not have the ADR verification job"),
    ("an accepted ADR whose verification step tolerates failure",
     [ACCEPTED, (CI, "        run: make evidence conformance-provenance\n",
                 "        run: make evidence conformance-provenance\n        continue-on-error: true\n")], ROOT, "CI does not have the ADR verification job"),
    ("an accepted ADR whose verification step has an if: after its run",
     [ACCEPTED, (CI, "        run: make evidence conformance-provenance\n",
                 "        run: make evidence conformance-provenance\n        if: false\n")], ROOT, "CI does not have the ADR verification job"),
    ("an accepted ADR whose verification job only dry-runs make",
     [ACCEPTED, (CI, "        run: make evidence conformance-provenance\n",
                 "        run: make -n evidence conformance-provenance\n")], ROOT, "CI does not have the ADR verification job"),
    ("a divergence added to the debt against the base",
     add_to_debt_against_the_base, ["sh", "scripts/debt-only-shrinks.sh", "HEAD~1"], "gained entries",
     unrelated_change_against_the_base),
    ("a debt base that does not resolve",
     lambda copy: against_a_base(copy, "no-such-commit"),
     ["sh", "-c", 'sh scripts/debt-only-shrinks.sh "$(cat base.txt)"'], "does not resolve to a commit",
     lambda copy: against_a_base(copy, "HEAD")),
    ("a debt with no base file, other than the one recorded",
     debt_without_a_base, ["sh", "scripts/debt-only-shrinks.sh", "HEAD~1"], "is not the debt ADR-001 recorded",
     debt_without_a_base_control),
    ("a debt added in the first commit of a push of two",
     debt_in_a_push_of_two, ["env", "GITHUB_EVENT_NAME=push", "GITHUB_EVENT_PATH=event.json",
                             "sh", "scripts/debt-only-shrinks.sh"], "gained entries",
     lambda copy: push_of_two(copy, "")),
    ("an accepted ADR whose workflow is not triggered by pull requests",
     [ACCEPTED, (CI, "  pull_request:\n", "  workflow_dispatch:\n")], ROOT, "CI does not have the ADR verification job"),
    ("an accepted ADR whose workflow sets defaults after its jobs",
     [ACCEPTED, (CI, APPEND, "defaults:\n  run:\n    shell: bash --noprofile --norc -c true {0}\n")],
     ROOT, "CI does not have the ADR verification job"),
    ("a skipped test named after a «### Verificación manual» heading",
     [(ADR, "## Contexto\n", "## Contexto\n\n### Verificación manual\n\nNinguna.\n"),
      ("conformance/conformance_test.go", "func TestConformance(t *testing.T) {\n",
       "func TestConformance(t *testing.T) {\n\tskipThroughAHelper(t)\n"),
      ("conformance/conformance_test.go", APPEND, "\nfunc skipThroughAHelper(t *testing.T) { t.Helper(); t.Skip(\"mutation\") }\n")],
     ["env", "CQL_EVIDENCE_NO_MUTATIONS=1", "sh", "scripts/evidence.sh"], "`TestConformance` skipped"),
    ("a byte of a vendored file changed",
     [("conformance/testdata/CqlTypesTest.xml", APPEND, "\n")], CONF, "CqlTypesTest.xml: git blob"),
    ("a file added to the suite",
     [("conformance/testdata/Extra.xml", WRITE, "<tests/>\n")], CONF, "Extra.xml is not in testdata/SOURCE.tsv"),
    ("a file removed from the suite",
     [("conformance/testdata/CqlTypesTest.xml", DELETE, None)], CONF, "CqlTypesTest.xml is in testdata/SOURCE.tsv and not"),
    ("a manifest entry listed twice",
     [("conformance/testdata/SOURCE.tsv", APPEND,
       "CqlTypesTest.xml\t0000000000000000000000000000000000000000\t" + "0" * 64 + "\n")],
     CONF, "lists CqlTypesTest.xml twice"),
    ("a reference run with another engine",
     [("testdata/snapshots/integer.java.json", '"engine" : "3.26.0"', '"engine" : "FAKE-0.0"')],
     ["go", "test", "-count=1", "-run", "TestTheReferencesAreOfTheirCorpus", "."], "the harness pins"),
    ("a classified divergence becomes unclassified",
     [("testdata/snapshots/integer.divergences.tsv", "D0\tjava\treference defect\t",
       "D0\tjava\tunclassified\t")],
     ["go", "test", "-count=1", "-run", "TestThePendingVerdictsDoNotGrow", "."],
     "has no verdict and is not in pending-verdicts.tsv"),
    ("debt swapped: one divergence classified, another unclassified in its place",
     [("testdata/snapshots/integer.divergences.tsv", "D0\tjava\treference defect\t",
       "D0\tjava\tunclassified\t"),
      ("testdata/snapshots/arithmetic.divergences.tsv", "D0\tjava\tunclassified\t", "D0\tjava\treference defect\t")],
     ["go", "test", "-count=1", "-run", "TestThePendingVerdictsDoNotGrow", "."],
     "has no verdict and is not in pending-verdicts.tsv"),
    ("an ADR names a test behind a build tag go test does not set",
     [("conformance/tagged_test.go", WRITE,
       "//go:build never\n\npackage conformance\n\nimport \"testing\"\n\nfunc TestGhostTagged(t *testing.T) {}\n"),
      (ADR, "`TestADRsHaveVerification`: este ADR", "`TestGhostTagged`: este ADR")],
     ROOT, "TestGhostTagged, which no package"),
    ("an ADR with a second verification section",
     [(ADR, APPEND, "\n## Verificación\n\n- `TestGhostInTheSecondSection`\n")], ROOT, "2 «## Verificación» sections"),
    ("the engine pin read from a comment",
     [("internal/referenceharness/pom.xml", "<artifactId>engine</artifactId>\n      <version>3.26.0</version>",
       "<artifactId>engine</artifactId>\n      <!-- <version>3.26.0</version> -->\n      <version>9.99.0</version>")],
     ["go", "test", "-count=1", "-run", "TestTheReferencesAreOfTheirCorpus", "."], 'the harness pins "9.99.0"'),
    ("a named test skipped through a helper, which only the evidence sees",
     [("conformance/conformance_test.go", "func TestConformance(t *testing.T) {\n",
       "func TestConformance(t *testing.T) {\n\tskipThroughAHelper(t)\n"),
      ("conformance/conformance_test.go", APPEND, "\nfunc skipThroughAHelper(t *testing.T) { t.Helper(); t.Skip(\"mutation\") }\n")],
     ["env", "CQL_EVIDENCE_NO_MUTATIONS=1", "sh", "scripts/evidence.sh"], "`TestConformance` skipped"),
]


if os.environ.get("CQL_MUTATIONS_NETWORK"):
    mutations = network_mutations


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
        edits(copy)
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
