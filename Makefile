MODULES := common overgate oversite
PACKAGES := ./common/... ./overgate/... ./oversite/...

.PHONY: build test vet fmt fmt-check check up down smoke integration

ACCEPTANCE_COMPOSE = docker compose -f compose.yaml -f compose.override.yaml -f compose.acceptance.yaml
.PHONY: acceptance-up acceptance-check acceptance-faults acceptance-down
acceptance-up:
	$(ACCEPTANCE_COMPOSE) up --build -d --wait --wait-timeout 180

acceptance-check:
	bash tests/proxyf/check.sh

acceptance-faults:
	bash tests/proxyf/faults.sh

acceptance-down:
	$(ACCEPTANCE_COMPOSE) down
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
