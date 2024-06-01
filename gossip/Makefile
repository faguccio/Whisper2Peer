.PHONY: format test staticcheck all

all: format test staticcheck checkUnion

format:
	go fmt ./...

test:
	go test -cover ./...

staticcheck:
	-staticcheck ./...

checkUnion:
	go-sumtype $(go list ./... | grep -v vendor)

test_coverage:
	go test -coverprofile=.cover -cover ./...
	go tool cover -html=.cover
