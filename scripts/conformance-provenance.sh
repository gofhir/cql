#!/bin/sh
# Provenance of the vendored conformance suite (docs/adr/001, docs/plans/002).
#   check            compare conformance/testdata/SOURCE.tsv with the git tree of
#                    cqframework/cql-tests at the recorded commit (needs network)
#   vendor <commit>  download tests/cql/*.xml at <commit> into conformance/testdata
#                    and rewrite SOURCE.tsv: the one way to re-vendor the suite
set -eu
repo=$(cd "$(dirname "$0")/.." && pwd)
python3 -I - "$repo" "$@" <<'PY'
import hashlib, json, os, shutil, sys, tempfile, urllib.request

repo, args = sys.argv[1], sys.argv[2:]
upstream, folder = "cqframework/cql-tests", "tests/cql/"
testdata = os.path.join(repo, "conformance", "testdata")
manifest = os.path.join(testdata, "SOURCE.tsv")


def get(url):
    req = urllib.request.Request(url, headers={"User-Agent": "gofhir-cql-provenance"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return r.read()


def tree(commit):
    data = json.loads(get(f"https://api.github.com/repos/{upstream}/git/trees/{commit}?recursive=1"))
    if data.get("truncated"):
        sys.exit("the upstream tree came back truncated")
    return {e["path"][len(folder):]: e["sha"] for e in data["tree"]
            if e["type"] == "blob" and e["path"].startswith(folder) and e["path"].endswith(".xml")
            and "/" not in e["path"][len(folder):]}


def blob(content):
    return hashlib.sha1(b"blob %d\0" % len(content) + content).hexdigest()


if args[:1] == ["vendor"] and len(args) == 2:
    commit = json.loads(get(f"https://api.github.com/repos/{upstream}/commits/{args[1]}"))["sha"]
    files = tree(commit)
    # Everything is downloaded and checked first: a failure halfway leaves the
    # suite and its manifest as they were.
    staging = tempfile.mkdtemp(prefix="cql-tests-")
    rows = []
    for name in sorted(files):
        content = get(f"https://raw.githubusercontent.com/{upstream}/{commit}/{folder}{name}")
        if blob(content) != files[name]:
            sys.exit(f"{name}: the download is not the blob the tree names; nothing was changed")
        open(os.path.join(staging, name), "wb").write(content)
        rows.append(f"{name}\t{files[name]}\t{hashlib.sha256(content).hexdigest()}")
    with open(os.path.join(staging, "SOURCE.tsv"), "w") as f:
        f.write(f"# repository: {upstream}\n# commit: {commit}\n# folder: {folder}\n")
        f.write("# file\tgit blob SHA-1\tSHA-256\n" + "\n".join(rows) + "\n")
    for old in os.listdir(testdata):
        if old.endswith(".xml"):
            os.remove(os.path.join(testdata, old))
    for name in os.listdir(staging):
        shutil.move(os.path.join(staging, name), os.path.join(testdata, name))
    os.rmdir(staging)
    print(f"vendored {len(rows)} files of {upstream}@{commit}")
elif args == ["check"]:
    meta, recorded = {}, {}
    for line in open(manifest):
        line = line.rstrip("\n")
        if line.startswith("# ") and ": " in line:
            k, v = line[2:].split(": ", 1)
            meta[k] = v
        elif line and not line.startswith("#"):
            name, sha1, _ = line.split("\t")
            recorded[name] = sha1
    files = tree(meta["commit"])
    problems = [f"{n}: recorded {recorded[n]}, upstream {files.get(n, 'absent')}" for n in sorted(recorded)
                if files.get(n) != recorded[n]]
    problems += [f"{n}: upstream has it, the manifest does not" for n in sorted(set(files) - set(recorded))]
    for p in problems:
        print(p)
    if problems:
        sys.exit(1)
    print(f"{len(recorded)} files are {upstream}@{meta['commit']}")
else:
    sys.exit("usage: conformance-provenance.sh check | vendor <commit>")
PY
