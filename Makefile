MATRIX_OS ?= darwin linux windows
MATRIX_ARCH ?= amd64

GIT_HASH ?= $(shell git show -s --format=%h)
GIT_TAG ?= $(shell git tag -l --merged $(GIT_HASH))
APP_VERSION ?= $(if $(TRAVIS_TAG),$(TRAVIS_TAG),$(if $(GIT_TAG),$(GIT_TAG),$(GIT_HASH)))
APP_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

REQ := $(patsubst assets/%,src/statuspage/%.go, $(wildcard assets/*))

-include artifacts/make/go/Makefile

artifacts/make/%/Makefile:
	curl -sf https://jmalloc.github.io/makefiles/fetch | bash /dev/stdin $*

.PHONY: install
install: vendor $(REQ) $(_SRC) | $(USE)
	$(eval PARTS := $(subst /, ,$*))
	$(eval BUILD := $(word 1,$(PARTS)))
	$(eval OS    := $(word 2,$(PARTS)))
	$(eval ARCH  := $(word 3,$(PARTS)))
	$(eval BIN   := $(word 4,$(PARTS)))
	$(eval ARGS  := $(if $(findstring debug,$(BUILD)),$(DEBUG_ARGS),$(RELEASE_ARGS)))

	CGO_ENABLED=$(CGO_ENABLED) GOOS="$(OS)" GOARCH="$(ARCH)" go install $(ARGS) "./src/cmd/..."

.PHONY: run
run: artifacts/build/debug/$(GOOS)/$(GOARCH)/proxy
	$< $(RUN_ARGS)

.PHOMY: docker
docker:
	docker build -t koshatul/auth-proxy:$(APP_VERSION) .

.PHONY: docker-local
docker-local: artifacts/build/release/linux/amd64/proxy
	docker build -t koshatul/auth-proxy:$(APP_VERSION) -f Dockerfile.local .

###
### Honeycomb targets
###

.PHONY: assets
assets: $(REQ)

MINIFY := $(GOPATH)/bin/minify
$(MINIFY):
	go get -u github.com/tdewolff/minify/cmd/minify

artifacts/assets/%.tmp: assets/% | $(MINIFY)
	@mkdir -p "$(@D)"
	$(MINIFY) -o "$@" "$<" || cp "$<" "$@"

src/statuspage/%.go: artifacts/assets/%.tmp
	@mkdir -p "$(@D)"
	@echo "package statuspage" > "$@"
	@echo 'const $(shell echo $(notdir $*) | tr [:lower:] [:upper:] | tr . _) = `' >> "$@"
	cat "$<" >> "$@"
	@echo '`' >> "$@"
