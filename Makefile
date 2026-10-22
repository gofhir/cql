# check is what has to pass before a change is done: the same as CI, plus the
# zones CI does not run and the oracle mutations of the open increments.
# references reruns the reference engines in Docker; it is separate because it
# needs Docker and the harness's dependencies (internal/referenceharness/README.md).

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
ZONES := UTC America/Santiago Asia/Kathmandu Pacific/Kiritimati Pacific/Midway Australia/Lord_Howe

.PHONY: check fmt lint test zones mutations references references-write conformance-provenance evidence

check: fmt lint test zones mutations

fmt:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt:"; echo "$$out"; exit 1; fi

lint:
	$(GOLANGCI_LINT) run ./...

test:
	go test -race -count=1 ./...

zones:
	@for z in $(ZONES); do echo "TZ=$$z"; TZ=$$z go test -count=1 ./... >/dev/null || exit 1; done

mutations:
	@for s in scripts/mutations/*.sh; do echo "== $$s"; sh $$s || exit 1; done

# conformance-provenance needs network: it compares conformance/testdata/SOURCE.tsv
# with the git tree of cqframework/cql-tests at the recorded commit, and runs the
# mutation only that comparison can catch.
conformance-provenance:
	sh scripts/conformance-provenance.sh check
	CQL_MUTATIONS_NETWORK=1 sh scripts/mutations/002-adr.sh

# evidence writes docs/evidence/<date>.md, the report increment records link.
evidence:
	sh scripts/evidence.sh

# references regenerates every reference file twice with Docker and checks the
# two runs are identical; references-write also copies them to testdata. The
# mutation that needs it runs here, not in make mutations.
references:
	sh scripts/references.sh check
	CQL_MUTATIONS_DOCKER=1 sh scripts/mutations/007-references.sh

references-write:
	sh scripts/references.sh write
