DOCKER_REPO := koshatul/auth-proxy

MATRIX_OS ?= darwin linux windows
MATRIX_ARCH ?= amd64

APP_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_HASH ?= $(shell git show -s --format=%h)

GO_DEBUG_ARGS   ?= -v -ldflags "-X main.version=$(GO_APP_VERSION)+debug -X main.gitHash=$(GIT_HASH) -X main.buildDate=$(APP_DATE)"
GO_RELEASE_ARGS ?= -v -ldflags "-X main.version=$(GO_APP_VERSION) -X main.gitHash=$(GIT_HASH) -X main.buildDate=$(APP_DATE) -s -w"

_GO_GTE_1_14 := $(shell expr `go version | cut -d' ' -f 3 | tr -d 'a-z' | cut -d'.' -f2` \>= 14)

ifeq "$(_GO_GTE_1_14)" "1"
_MODFILEARG := -modfile tools.mod
endif


-include .makefiles/Makefile
-include .makefiles/pkg/docker/v1/Makefile
-include .makefiles/pkg/go/v1/Makefile

.makefiles/%:
	@curl -sfL https://makefiles.dev/v1 | bash /dev/stdin "$@"

.PHONY: install
install: vendor $(REQ) $(_SRC) | $(USE)
	$(eval PARTS := $(subst /, ,$*))
	$(eval BUILD := $(word 1,$(PARTS)))
	$(eval OS    := $(word 2,$(PARTS)))
	$(eval ARCH  := $(word 3,$(PARTS)))
	$(eval BIN   := $(word 4,$(PARTS)))
	$(eval ARGS  := $(if $(findstring debug,$(BUILD)),$(DEBUG_ARGS),$(RELEASE_ARGS)))

	CGO_ENABLED=$(CGO_ENABLED) GOOS="$(OS)" GOARCH="$(ARCH)" go install $(ARGS) "./cmd/..."

.PHONY: run
run: artifacts/build/debug/$(GOHOSTOS)/$(GOHOSTARCH)/proxy
	"$<" $(RUN_ARGS)

# .PHONY: docker
# docker:
# 	docker build -t koshatul/auth-proxy:$(APP_VERSION) .

.PHONY: docker-local
docker-local: artifacts/build/release/linux/amd64/proxy
	docker build -t koshatul/auth-proxy:$(APP_VERSION) -f Dockerfile.local .

.PHONY: docker-test
docker-test:
	docker run -ti --rm -v $(shell pwd):/go/src/github.com/koshatul/auth-proxy --workdir /go/src/github.com/koshatul/auth-proxy golang:1.14 make

######################
# Testing
######################

GINKGO := artifacts/ginkgo/bin/ginkgo
$(GINKGO):
	@mkdir -p "$(MF_PROJECT_ROOT)/$(@D)"
	GOBIN="$(MF_PROJECT_ROOT)/$(@D)" go get $(_MODFILEARG) github.com/onsi/ginkgo/ginkgo

test:: $(GINKGO)
	-@mkdir -p "artifacts/test"
	$(GINKGO) -outputdir "artifacts/test/" -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --compilers=3 --nodes=6

######################
# Linting
######################

MISSPELL := artifacts/misspell/bin/misspell
$(MISSPELL):
	@mkdir -p "$(MF_PROJECT_ROOT)/$(@D)"
	GOBIN="$(MF_PROJECT_ROOT)/$(@D)" go get $(_MODFILEARG) github.com/client9/misspell/cmd/misspell

GOLINT := artifacts/golint/bin/golint
$(GOLINT):
	@mkdir -p "$(MF_PROJECT_ROOT)/$(@D)"
	GOBIN="$(MF_PROJECT_ROOT)/$(@D)" go get $(_MODFILEARG) golang.org/x/lint/golint

GOLANGCILINT := artifacts/golangci-lint/bin/golangci-lint
$(GOLANGCILINT):
	@mkdir -p "$(MF_PROJECT_ROOT)/$(@D)"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(MF_PROJECT_ROOT)/$(@D)" v1.23.8

STATICCHECK := artifacts/staticcheck/bin/staticcheck
$(STATICCHECK):
	@mkdir -p "$(MF_PROJECT_ROOT)/$(@D)"
	GOBIN="$(MF_PROJECT_ROOT)/$(@D)" go get $(_MODFILEARG) honnef.co/go/tools/cmd/staticcheck

.PHONY: lint
lint:: $(MISSPELL) $(GOLINT) $(GOLANGCILINT) $(STATICCHECK)
	go vet ./...
	$(GOLINT) -set_exit_status ./...
	$(MISSPELL) -w -error -locale UK ./...
	$(GOLANGCILINT) run --enable-all --disable=gomnd ./...
	$(STATICCHECK) -checks all -fail "all,-U1001" -unused.whole-program ./...
