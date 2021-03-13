
.PHONY: test
test: 
	go test -v -covermode=atomic -cover -race -coverprofile=coverage.txt -p 1 ./...

.PHONY: lint
lint:
	#
	# this will run the default linters on non-test files
	# and then all but the "errcheck" linters on the tests
	golangci-lint run --tests=false --exclude-use-default
	golangci-lint run -D=errcheck   --exclude-use-default

.PHONY: checks
checks:
	go vet ./...
	echo "checking gofmt"
	echo " ! gofmt -d *.go       2>&1 | read " | bash
	echo "checking gofmt - DONE"

.PHONY: upload-coverage
upload-coverage:
	go get github.com/mattn/goveralls
	/go/bin/goveralls -coverprofile=coverage.txt -service=drone.io
