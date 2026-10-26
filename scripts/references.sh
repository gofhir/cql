#!/bin/sh
# Regenerates every reference file the tests compare against, with the
# versions the harness pins (internal/referenceharness/pom.xml and
# js/package-lock.json), twice and in two directories: the two runs have to be
# identical byte for byte, or a reference holds something of the run and not
# of the answer (plan 007). Needs Docker.
#
#   sh scripts/references.sh check   regenerates, and fails listing the files that differ from testdata
#   sh scripts/references.sh write   regenerates, and copies them to testdata
set -eu
mode=${1:-check}
case $mode in check | write) ;; *)
	echo "usage: sh scripts/references.sh check|write" >&2
	exit 2
	;;
esac
repo=$(cd "$(dirname "$0")/.." && pwd)
harness=$repo/internal/referenceharness
work=$(mktemp -d "${TMPDIR:-/tmp}/references.XXXXXX")
trap 'rm -rf "$work"' EXIT

# The images the harness runs in, each at an exact version: a tag that moves
# would change the references under both runs alike, where their comparison
# cannot see it. The latest LTS of each (plan 007).
java_image=maven:3.10.0-eclipse-temurin-25
node_image=node:24.22.1

areas="arithmetic integer scope types fhir codes codesets membership fhirmembership division units temporal crossmodel"

# The harness, built from what it pins.
docker run --rm -v "$harness":/w -w /w "$java_image" sh -c \
	'rm -rf lib out && mvn -q -B dependency:copy-dependencies -DoutputDirectory=/w/lib && mkdir out && javac -cp "lib/*" -d out src/*.java'
docker run --rm -v "$harness/js":/w -w /w "$node_image" npm ci --silent

# The way the harness writes a FHIR value 5.x hands it, held to the value over
# fhir-data.json and the official R4 examples (RoundTrip.java), before it
# writes any: the package at its SHA-256, or the check is of another one.
examples_sha256=e18b31e7a52145a31f1f3f409cf6847583b08b7306cc4e9460a95a7b5efba930
mkdir "$work/examples"
curl -fsSL -o "$work/examples/package.tgz" https://packages.fhir.org/hl7.fhir.r4.examples/4.0.1
if [ "$(shasum -a 256 "$work/examples/package.tgz" | cut -d' ' -f1)" != "$examples_sha256" ]; then
	echo "hl7.fhir.r4.examples 4.0.1 is not the package pinned" >&2
	exit 1
fi
tar xzf "$work/examples/package.tgz" -C "$work/examples"
cp "$repo/testdata/snapshots/fhir-data.json" "$work/examples/"
docker run --rm -v "$harness":/w -v "$work/examples":/x -w /w "$java_image" \
	java -cp "out:lib/*" RoundTrip /x/fhir-data.json /x/package

# generate writes every reference into $1/testdata, at the path the tests read
# it from, and the ELM it needs on the way into $1/elm.
generate() {
	out=$1
	mkdir -p "$out/in/lib" "$out/elm" "$out/testdata/snapshots" "$out/testdata/reference" \
		"$out/testdata/membership" "$out/testdata/membershipmodel"
	cp "$repo/fhirhelpers/FHIRHelpers-4.0.1.cql" "$out/in/lib/"
	cp -R "$repo/testdata/snapshots" "$repo/testdata/reference" "$repo/testdata/membership" \
		"$repo/testdata/membershipmodel" "$out/in/"
	cp "$repo/fhirhelpers/FHIRHelpers-4.0.1.cql" "$out/in/reference/"
	docker run --rm -v "$harness":/w -v "$out":/o -w /w -e "AREAS=$areas" "$java_image" sh -ec '
		java -cp "out:lib/*" Translate /o/in/lib/FHIRHelpers-4.0.1.cql /o/in/lib > /o/elm/FHIRHelpers.elm.json
		for a in $AREAS; do
			data=-
			case $a in fhir | fhirmembership | crossmodel) data=/o/in/snapshots/fhir-data.json ;; esac
			java -cp "out:lib/*" Evaluate -each /o/in/snapshots/$a.cql /o/in/lib $data /o/elm/$a.elm.json \
				> /o/testdata/snapshots/$a.java.json
		done
		java -cp "out:lib/*" Evaluate -each /o/in/reference/encoder.cql /o/in/lib /o/in/snapshots/fhir-data.json \
			> /o/testdata/reference/encoder.java.json
		java -cp "out:lib/*" Translate /o/in/reference/probe.cql /o/in/reference > /o/elm/probe.elm.json
		java -cp "out:lib/*" Evaluate /o/in/reference/runtime.cql /o/in/reference > /o/testdata/reference/runtime.expected.json
		java -cp "out:lib/*" Translate /o/in/reference/runtime.cql /o/in/reference > /o/elm/runtime.elm.json
		java -cp "out:lib/*" Translate /o/in/membership/cases.cql /o/in/membership > /o/testdata/membership/cases.elm.json
		for d in absent present; do
			java -cp "out:lib/*" Evaluate -each /o/in/membershipmodel/cases.cql /o/in/lib /o/in/membershipmodel/$d.json \
				/o/testdata/membershipmodel/cases.elm.json > /o/testdata/membershipmodel/$d.java.json
		done'
	docker run --rm -v "$harness":/w -v "$out":/o -w /w -e "AREAS=$areas" "$node_image" sh -ec '
		for a in $AREAS; do
			data=
			case $a in fhir | fhirmembership | crossmodel) data=/o/in/snapshots/fhir-data.json ;; esac
			node js/evaluate.js --each /o/elm/$a.elm.json /o/in/snapshots/$a.cql /o/elm/FHIRHelpers.elm.json $data \
				> /o/testdata/snapshots/$a.js.json
		done
		node js/evaluate.js /o/elm/runtime.elm.json /o/in/reference/runtime.cql > /o/testdata/reference/runtime.expected.js.json'
	(cd "$repo" && go run ./internal/referenceharness/differential/nodes -cql testdata/snapshots/fhirmembership.cql \
		-elm "$out/elm/fhirmembership.elm.json" > "$out/testdata/snapshots/fhirmembership.nodes.tsv")
	(cd "$repo" && go run ./internal/referenceharness/fold -in "$out/elm/probe.elm.json" -source probe.cql \
		-out "$out/testdata/reference/probe.expected.json")
}

generate "$work/a"
generate "$work/b"
if ! diff -r "$work/a/testdata" "$work/b/testdata" > "$work/runs.diff"; then
	echo "two runs of the harness differ:" >&2
	head -40 "$work/runs.diff" >&2
	exit 1
fi
echo "two runs identical: $(cd "$work/a" && find testdata -type f | wc -l | tr -d ' ') files"

changed=0
for f in $(cd "$work/a" && find testdata -type f | sort); do
	if ! cmp -s "$work/a/$f" "$repo/$f"; then
		echo "differs: $f"
		changed=$((changed + 1))
	fi
done
# A reference in the repository the harness no longer writes is stale too.
for f in $(cd "$repo" && ls testdata/snapshots/*.java.json testdata/snapshots/*.js.json testdata/snapshots/*.nodes.tsv \
	testdata/reference/*.java.json testdata/reference/*.expected*.json testdata/membership/*.elm.json \
	testdata/membershipmodel/*.elm.json testdata/membershipmodel/*.java.json); do
	if [ ! -f "$work/a/$f" ]; then
		echo "not written by the harness: $f"
		changed=$((changed + 1))
	fi
done
echo "$changed files differ from the repository"
if [ "$mode" = write ]; then
	(cd "$work/a" && find testdata -type f) | while read -r f; do cp "$work/a/$f" "$repo/$f"; done
	echo "written"
elif [ "$changed" -gt 0 ]; then
	exit 1
fi
