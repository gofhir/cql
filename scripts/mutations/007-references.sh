#!/bin/sh
# Mutations of the controls plan 007 puts in before the references move to
# their latest version (docs/plans/007-referencias-al-dia.md, PR 1): the engine
# held to each oracle over the whole corpus, the encoder of the Java harness,
# the messages the references record, and the drift report. Each is applied
# to a copy of the repository; the command it names has to fail, and its output
# has to say why. The one that needs Docker runs only with CQL_MUTATIONS_DOCKER=1.
# Usage: sh scripts/mutations/007-references.sh [work-directory]
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
if [ $# -ge 1 ]; then
    work=$1
else
    work=$(mktemp -d "${TMPDIR:-/tmp}/mutations-007.XXXXXX")
    trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT
fi
python3 -I - "$repo" "$work" <<'PY'
import os, shutil, subprocess, sys

repo, work = sys.argv[1], sys.argv[2]
files = subprocess.run(["git", "-C", repo, "ls-files", "-co", "--exclude-standard"],
                       capture_output=True, text=True, check=True).stdout.split()

OWN = "engine_own_defects_test.go"
ENCODER = "engine_reference_encoder_test.go"
DRIFT = "internal/referenceharness/differential/drift/main.go"


def run(*names):
    return ["go", "test", "-count=1", "-run", "^(" + "|".join(names) + ")$", "."]


FOLLOWS = run("TestTheEngineFollowsTheOracles")
ENCODES = run("TestTheReferenceEncoderWritesEachValueAsItIs")
MESSAGES = run("TestTheReferenceMessagesAreOfTheAnswer")
DRIFTS = ["go", "test", "-count=1", "./internal/referenceharness/differential/drift/"]
LONG_TIME = "Cannot cast a value of type java.lang.Long as org.opencds.cqf.cql.engine.runtime.Time."

# name, (path, old, new), command, text the failure has to hold
mutations = [
    # The engine against each oracle, without the references.
    ("an own defect dropped from its file",
     ("testdata/snapshots/fhir.own-defects.tsv", "\nD19\t", "\nD19-gone\t"), FOLLOWS, "does not record it"),
    ("an own defect recorded with another answer of the engine",
     ("testdata/snapshots/arithmetic.own-defects.tsv", "\nD427\tnull\tDecimal 2688", "\nD427\tnull\tDecimal 2788"),
     FOLLOWS, "recorded Decimal 2788"),
    ("the fhir oracle finds no defect",
     (OWN, "\t\tif engine[name] != v.expected {", "\t\tif false && engine[name] != v.expected {"), FOLLOWS, "no longer finds"),
    ("the arithmetic oracle reads every answer the other way",
     (OWN, "\t\tif !follows(readEngineArithmetic(engine[name]), v) {", "\t\tif follows(readEngineArithmetic(engine[name]), v) {"),
     FOLLOWS, "FAIL"),
    ("the engine adds Decimals as a subtraction",
     ("eval/evaluator.go", "funcs.ScaleDecimal(ld.Add(rd))", "funcs.ScaleDecimal(ld.Sub(rd))"), FOLLOWS, "arithmetic"),
    # The encoder of the Java harness.
    ("a case of the encoder written another way",
     (ENCODER, '"D16": `{"type": "Code", "code": "c", "system": null,', '"D16": `{"type": "Code", "code": "c", "system": "",'),
     ENCODES, "D16"),
    ("an extension the harness lost",
     ("testdata/reference/encoder.java.json", '{\\"url\\":\\"http://example.org/e\\",', '{\\"url\\":\\"http://example.org/other\\",'),
     ENCODES, "D22"),
    ("the encoder's output of another corpus",
     ("testdata/reference/encoder.cql", "define D0: true\n", "define D0: true\n// edited\n"), ENCODES, "another encoder.cql"),
    # The messages the references record.
    ("a Java object's identity in a message",
     ("testdata/snapshots/types.java.json", LONG_TIME, "Cannot cast org.hl7.elm.r1.NamedTypeSpecifier@21e57990."),
     MESSAGES, "Java object's identity"),
    ("an error with no message",
     ("testdata/snapshots/types.java.json", LONG_TIME, ""), MESSAGES, "no message"),
    ("an error whose message is no text",
     ("testdata/snapshots/types.java.json", '"' + LONG_TIME + '"', '{"unencoded": "x"}'), MESSAGES, "no text"),
    # The drift report.
    ("drift reads a changed message as no change",
     (DRIFT, "\tcase old.message != cur.message:\n\t\treturn messageOnly", "\tcase old.message != cur.message:\n\t\treturn same"),
     DRIFTS, "FAIL"),
    ("drift ignores a definition only the new set answers",
     (DRIFT, "\tfor n := range cur {\n\t\tnames[n] = true\n\t}\n", ""), DRIFTS, "FAIL"),
    ("drift keeps a reclassified row as kept",
     (DRIFT, "\t\tcase now != class:", "\t\tcase false:"), DRIFTS, "FAIL"),
    ("drift lists no row that moved",
     (DRIFT, "\t\tif old[k] == cur[k] {\n\t\t\tcontinue", "\t\tif true {\n\t\t\tcontinue"), DRIFTS, "FAIL"),
    ("drift reads a refusal that now errs at run time as a value",
     (DRIFT, "\t\treturn acceptedNow // the translator accepts it now", "\t\treturn errorToValue // the translator accepts it now"),
     DRIFTS, "FAIL"),
]
if os.environ.get("CQL_MUTATIONS_DOCKER") == "1":
    # The reference files depend on the run: two runs of the harness differ.
    mutations.append(("a reference that holds the time of its run",
        ("internal/referenceharness/src/Evaluate.java", 'out.put("results", results);\n        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));\n    }\n\n    /** Una definición',
         'out.put("results", results);\n        out.put("ranAt", System.nanoTime());\n        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));\n    }\n\n    /** Una definición'),
        ["sh", "scripts/references.sh", "check"], "two runs of the harness differ"))
    # A reference in the repository other than the one the harness writes.
    mutations.append(("a reference other than the harness writes",
        ("testdata/snapshots/codes.js.json", '"D0":', '"D0" :'), ["sh", "scripts/references.sh", "check"],
        "1 files differ from the repository"))
    # A reference in the repository the harness no longer writes.
    mutations.append(("a reference the harness no longer writes",
        ("scripts/references.sh", 'areas="arithmetic integer', 'areas="integer'), ["sh", "scripts/references.sh", "check"],
        "not written by the harness: testdata/snapshots/arithmetic.java.json"))

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
