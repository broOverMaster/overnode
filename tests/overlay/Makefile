# Пути определяются относительно этого файла, независимо от текущего каталога.
ACCEPTANCE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ACCEPTANCE_COMPOSE = docker compose -f "$(ACCEPTANCE_DIR)compose.yaml"

.PHONY: acceptance-up acceptance-check acceptance-faults acceptance-down
acceptance-up:
	$(ACCEPTANCE_COMPOSE) up --build -d --wait --wait-timeout 180

acceptance-check:
	bash "$(ACCEPTANCE_DIR)check.sh"

acceptance-faults:
	bash "$(ACCEPTANCE_DIR)faults.sh"

acceptance-down:
	$(ACCEPTANCE_COMPOSE) down
