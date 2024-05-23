.PHONY: format test staticcheck all

all: format test staticcheck

format:
	go fmt ./...

test:
	go test -cover ./...

staticcheck:
	-staticcheck ./...

test_coverage:
	go test -coverprofile=.cover -cover ./...
	go tool cover -html=.cover
