.PHONY: build clean test scan

install-dev:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go get -u github.com/canthefason/go-watcher
	
# Quality rules

test: ## Run unittests
	go test ./...
test-out:
	mkdir -p report
	go test -json ./... | tee report/test.out

coverage: ## Describes how much of a package's code is exercised by running the package's tests
	go test ./... -json -cover
coverage-out:
	mkdir -p report
	go test ./... -json -cover -coverprofile=report/coverage.out
coverage-html-out:
	go tool cover -html=report/coverage.out
vet: ## Reports suspicious constructs
	go vet ./...
vet-out:
	mkdir -p report
	go vet ./... 2>&- | tee report/govet.out

staticcheck: ## Reports static analysis, it finds bugs and performance issues.
	staticcheck -checks=all ./...
staticcheck-out:
	mkdir -p report
	staticcheck -checks=all ./... | tee report/staticcheck.out

scan: test-out coverage-out vet-out staticcheck-out
scan-cli: test coverage vet staticcheck

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
