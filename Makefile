# check is what has to pass before a change is done: the same as CI, plus the
# zones CI does not run and the oracle mutations of the open increments.
# references reruns the reference engines in Docker; it is separate because it
# needs Docker and the harness's dependencies (internal/referenceharness/README.md).

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
ZONES := UTC America/Santiago Asia/Kathmandu Pacific/Kiritimati Pacific/Midway Australia/Lord_Howe

.PHONY: check fmt lint test zones mutations references

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
	sh scripts/mutations/001-decimal-division.sh

references:
	@echo "see internal/referenceharness/README.md: the commands depend on the area"
	@exit 1
