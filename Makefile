MODULES := common overgate oversite
PACKAGES := ./common/... ./overgate/... ./oversite/...

.PHONY: build test vet fmt fmt-check check up down smoke integration

.DEFAULT_GOAL := build

build:
	mkdir -p .local/bin
	go build -trimpath -o .local/bin/overgate ./overgate/cmd/overgate
	go build -trimpath -o .local/bin/oversite ./oversite/cmd/oversite

test:
	go test $(PACKAGES)

vet:
	go vet $(PACKAGES)

fmt:
	gofmt -w $(MODULES)

fmt-check:
	@test -z "$$(gofmt -l $(MODULES))" || { gofmt -l $(MODULES); exit 1; }

check: fmt-check vet test build

up:
	docker compose up --build -d

down:
	docker compose down
