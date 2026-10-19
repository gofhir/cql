#!/bin/sh
# The unclassified divergences only shrink (docs/adr/001, docs/plans/002): the
# diff of testdata/snapshots/pending-verdicts.tsv against the base adds no
# line. A test of the repository cannot hold that, since one change can edit
# both the manifests and the debt; the base it is compared against can.
#
# The base is the first argument; in CI, without one, the pull request's
# merge-base with its base branch, or the commit a push started from. It is
# never read from a variable a workflow could set. A base that does not resolve
# fails: the comparison fails closed.
set -eu
debt=testdata/snapshots/pending-verdicts.tsv
# The debt as ADR-001 recorded it: the only content accepted where the base
# does not have the file yet, the change that introduces it.
initial_sha256=2655bb9f9273fff0c9953bbc225b7f8df70fbd3a0b26934afd284f124908f100

if [ $# -ge 1 ]; then
    base=$1
elif [ -n "${GITHUB_BASE_REF:-}" ]; then
    git fetch --quiet origin "$GITHUB_BASE_REF"
    base=$(git merge-base HEAD "origin/$GITHUB_BASE_REF")
elif [ "${GITHUB_EVENT_NAME:-}" = push ] && [ -n "${GITHUB_EVENT_PATH:-}" ]; then
    base=$(python3 -I -c 'import json, sys; print(json.load(open(sys.argv[1]))["before"])' "$GITHUB_EVENT_PATH")
else
    echo "no base to compare $debt with: pass one, or run in CI"
    exit 1
fi
if ! git rev-parse --quiet --verify "$base^{commit}" >/dev/null; then
    echo "the base $base does not resolve to a commit: nothing proves the debt did not grow"
    exit 1
fi
if ! git cat-file -e "$base:$debt" 2>/dev/null; then
    got=$(shasum -a 256 "$debt" 2>/dev/null | cut -d' ' -f1 || sha256sum "$debt" | cut -d' ' -f1)
    if [ "$got" = "$initial_sha256" ]; then
        echo "$debt is introduced here, as ADR-001 recorded it"
        exit 0
    fi
    echo "$debt is not at $base, and is not the debt ADR-001 recorded: nothing proves it did not grow"
    exit 1
fi
added=$(git diff --unified=0 "$base" -- "$debt" | grep '^+' | grep -v '^+++' || true)
if [ -n "$added" ]; then
    echo "$debt gained entries against $base; a divergence has to be classified, not added to the debt:"
    echo "$added"
    exit 1
fi
echo "$debt only shrinks against $base"
