ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
EXCLUDE_DIRS := ./loadtest
EXCLUDE_GO_MOD_DIRS := $(shell find $(EXCLUDE_DIRS) -type f -name 'go.mod' -exec dirname {} \; | sort)

.PHONY: all
all: fmt lint test

setup:
	mkdir -p $(GOPATH)/bin
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@7c5a9af06 # TODO: bump after newer release

lint:
	@$(call run-go-mod-dir, golangci-lint run, Linting)

fmt:
	@$(call run-go-mod-dir, golangci-lint fmt, Formatting)

test:
	@$(call run-go-mod-dir-exclude, go test -race -timeout 30s, Testing)
	@$(call run-go-mod-dir-exclude, go test -race -timeout 30s -tags synctest, "Testing synctest")

gomod.tidy:
	@$(call run-go-mod-dir, go mod tidy, "Tidying go.mod")

# run-go-mod-dir runs the given $1 command in all the directories with
# a go.mod file
define run-go-mod-dir
set -e; \
for dir in $(ALL_GO_MOD_DIRS); do \
	[ -z $(2) ] || echo "$(2) $${dir}/..."; \
	cd "$(PROJECT_DIR)/$${dir}" && PATH=$(BIN_DIR):$$PATH $(1); \
done;
endef

# run-go-mod-dir-exclude runs the given $1 command in all the directories with
# a go.mod file except the directories in EXCLUDE_DIRS
define run-go-mod-dir-exclude
set -e; \
for dir in $(filter-out $(EXCLUDE_GO_MOD_DIRS),$(ALL_GO_MOD_DIRS)); do \
	[ -z $(2) ] || echo "$(2) $${dir}/..."; \
	cd "$(PROJECT_DIR)/$${dir}" && PATH=$(BIN_DIR):$$PATH $(1); \
done;
endef