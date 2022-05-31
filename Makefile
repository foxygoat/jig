# Run `make help` to display help

# --- Global -------------------------------------------------------------------
O = out
COVERAGE = 0
VERSION ?= $(shell git describe --tags --dirty  --always)
REPO_ROOT = $(shell git rev-parse --show-toplevel)

all: build test check-coverage lint lint-proto ## build, test, check coverage and lint
	@if [ -e .git/rebase-merge ]; then git --no-pager log -1 --pretty='%h %s'; fi
	@echo '$(COLOUR_GREEN)Success$(COLOUR_NORMAL)'

ci: clean check-uptodate all  ## Full clean build and up-to-date checks as run on CI

check-uptodate: proto golden tidy
	test -z "$$(git status --porcelain -- go.mod go.sum '*.pb' '*.jsonnet' '*.js')" || { git status; false; }

clean::  ## Remove generated files
	-rm -rf $(O)

.PHONY: all check-uptodate ci clean

# --- Build --------------------------------------------------------------------
GO_MODULE := $(shell go list -m)
GO_LDFLAGS = -X main.version=$(VERSION)
CMDS = . ./internal/cmd/client ./internal/cmd/server

build: | $(O)  ## Build reflect binaries
	go build -o $(O) -ldflags='$(GO_LDFLAGS)' $(CMDS)

install:  ## Build and install binaries in $GOBIN
	go install -ldflags='$(GO_LDFLAGS)' $(CMDS)

tidy:  ## Tidy go modules with "go mod tidy"
	go mod tidy

.PHONY: build install tidy

# --- Test ---------------------------------------------------------------------
COVERFILE = $(O)/coverage.txt

test: | $(O) ## Run tests and generate a coverage file
	go test -coverprofile=$(COVERFILE) ./...

check-coverage: test  ## Check that test coverage meets the required level
	@go tool cover -func=$(COVERFILE) | $(CHECK_COVERAGE) || $(FAIL_COVERAGE)

cover: test  ## Show test coverage in your browser
	go tool cover -html=$(COVERFILE)

RUN_BONES = $(O)/jig bones --force --language=$(1) --quote-style=$(2) --proto-set pb/$(3)/$(4) --minimal=$(5) --method-dir bones/testdata/golden/$(3)-$(2)-$(5)-minimal
golden: build proto  ## Generate golden test files
	$(call RUN_BONES,jsonnet,double,exemplar,exemplar.pb,no)
	$(call RUN_BONES,jsonnet,double,exemplar,exemplar.pb,yes)
	$(call RUN_BONES,jsonnet,double,greet,greeter.pb,no)
	$(call RUN_BONES,jsonnet,double,greet,greeter.pb,yes)
	$(call RUN_BONES,jsonnet,single,exemplar,exemplar.pb,no)
	$(call RUN_BONES,jsonnet,single,exemplar,exemplar.pb,yes)
	$(call RUN_BONES,jsonnet,single,greet,greeter.pb,no)
	$(call RUN_BONES,jsonnet,single,greet,greeter.pb,yes)
	$(call RUN_BONES,js,double,exemplar,exemplar.pb,no)
	$(call RUN_BONES,js,double,exemplar,exemplar.pb,yes)
	$(call RUN_BONES,js,double,greet,greeter.pb,no)
	$(call RUN_BONES,js,double,greet,greeter.pb,yes)
	$(call RUN_BONES,js,single,exemplar,exemplar.pb,no)
	$(call RUN_BONES,js,single,exemplar,exemplar.pb,yes)
	$(call RUN_BONES,js,single,greet,greeter.pb,no)
	$(call RUN_BONES,js,single,greet,greeter.pb,yes)

CHECK_COVERAGE = awk -F '[ \t%]+' '/^total:/ {print; if ($$3 < $(COVERAGE)) exit 1}'
FAIL_COVERAGE = { echo '$(COLOUR_RED)FAIL - Coverage below $(COVERAGE)%$(COLOUR_NORMAL)'; exit 1; }

.PHONY: check-coverage cover golden test

# --- Lint ---------------------------------------------------------------------

lint:  ## Lint go source code
	golangci-lint run

.PHONY: lint

# --- Protos ---------------------------------------------------------------------
PROTOFILES = $(shell find proto -regex '.*/[^_][^/]*\.proto' -print | LANG=C sort)

lint-proto:  ## Lint *.proto files
	buf lint proto

proto:  ## Generate Go pb and grpc bindings and FileDescritor set for .proto files
	$(foreach PROTO,$(PROTOFILES),$(GENPROTO)$(nl))
	gofumpt -w pb
	cp pb/greet/greeter.pb pb/google/protobuf/duration.pb serve/testdata/greet
	cp pb/httpgreet/httpgreet.pb serve/testdata/httpgreet

.PHONY: lint-proto proto

# GENPROTO is called with $(PROTO) set by "foreach" to the filename of the input proto.
define GENPROTO
	protoc \
		-I proto \
		$(GENPROTO_PB_FLAGS) \
		$(if $(SHOULD_GEN_GO),$(GENPROTO_GO_FLAGS)) \
		$(PROTO)
endef

# Only generate Go bindings if the proto file declares a go_package that
# starts with our module prefix. SHOULD_GEN_GO will be a non-empty string
# in this case.
SHOULD_GEN_GO = $(shell grep -l '^option go_package = "$(GO_MODULE)[/"]' $(PROTO))
GENPROTO_GO_FLAGS = --go_out=paths=source_relative:pb --go-grpc_out=paths=source_relative:pb
GENPROTO_PB_FLAGS = --descriptor_set_out=$(PROTO:proto/%.proto=pb/%.pb) --include_imports

# --- Release -------------------------------------------------------------------
release: nexttag  ## Tag and release binaries for different OS on GitHub release
	git tag $(NEXTTAG)
	git push origin $(NEXTTAG)
	goreleaser release --rm-dist

nexttag:
	$(eval NEXTTAG := $(shell $(NEXTTAG_CMD)))

.PHONY: nexttag release

define NEXTTAG_CMD
{ git tag --list --merged HEAD --sort=-v:refname; echo v0.0.0; }
| grep -E "^v?[0-9]+.[0-9]+.[0-9]+$$"
| head -n1
| awk -F . '{ print $$1 "." $$2 "." $$3 + 1 }'
endef

# --- Utilities ----------------------------------------------------------------
COLOUR_NORMAL = $(shell tput sgr0 2>/dev/null)
COLOUR_RED    = $(shell tput setaf 1 2>/dev/null)
COLOUR_GREEN  = $(shell tput setaf 2 2>/dev/null)
COLOUR_WHITE  = $(shell tput setaf 7 2>/dev/null)

help:
	@awk -F ':.*## ' 'NF == 2 && $$1 ~ /^[A-Za-z0-9%_-]+$$/ { printf "$(COLOUR_WHITE)%-25s$(COLOUR_NORMAL)%s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

$(O):
	@mkdir -p $@

.PHONY: help

define nl


endef
ifndef ACTIVE_HERMIT
$(eval $(subst \n,$(nl),$(shell bin/hermit env -r | sed 's/^\(.*\)$$/export \1\\n/')))
endif

# Ensure make version is gnu make 3.82 or higher
ifeq ($(filter undefine,$(value .FEATURES)),)
$(error Unsupported Make version. \
	$(nl)Use GNU Make 3.82 or higher (current: $(MAKE_VERSION)). \
	$(nl)Activate 🐚 hermit with `. bin/activate-hermit` and run again \
	$(nl)or use `bin/make`)
endif
