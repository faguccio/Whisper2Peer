.PHONY: format test staticcheck all mrproper generate build

all: format test staticcheck checkUnion build

format:
	go fmt ./...

test:
	go test -cover ./...

staticcheck:
	-staticcheck ./...

generate:
	-go generate ./...

checkUnion:
	go-sumtype $$(go list ./... | grep -v vendor)

test_coverage:
	go test -coverprofile=.cover -cover ./...
	go tool cover -html=.cover

mrproper:
	$(RM) ./horizontalAPI/types/*.go

build:
	go build -o gossip ./cmd/main
	go build ./cmd/testing
