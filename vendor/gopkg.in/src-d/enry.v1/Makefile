# Package configuration
PROJECT = enry
COMMANDS = cmd/enry

# Including ci Makefile
MAKEFILE = Makefile.main
CI_REPOSITORY = https://github.com/src-d/ci.git
CI_FOLDER = .ci
$(MAKEFILE):
	@git clone --quiet $(CI_REPOSITORY) $(CI_FOLDER); \
	cp $(CI_FOLDER)/$(MAKEFILE) .;
-include $(MAKEFILE)

# Docsrv: configure the languages whose api-doc can be auto generated
LANGUAGES = go
# Docs: do not edit this
DOCS_REPOSITORY := https://github.com/src-d/docs
SHARED_PATH ?= $(shell pwd)/.docsrv-resources
DOCS_PATH ?= $(SHARED_PATH)/.docs
$(DOCS_PATH)/Makefile.inc:
	git clone --quiet --depth 1 $(DOCS_REPOSITORY) $(DOCS_PATH);
-include $(DOCS_PATH)/Makefile.inc

LINGUIST_PATH = .linguist

# build CLI
LOCAL_TAG := $(shell git describe --tags --abbrev=0)
LOCAL_COMMIT := $(shell git rev-parse --short HEAD)
LOCAL_BUILD := $(shell date +"%m-%d-%Y_%H_%M_%S")
LOCAL_LDFLAGS = -s -X main.version=$(LOCAL_TAG) -X main.build=$(LOCAL_BUILD) -X main.commit=$(LOCAL_COMMIT)

# shared objects
RESOURCES_DIR=./.shared
LINUX_DIR=$(RESOURCES_DIR)/linux-x86-64
LINUX_SHARED_LIB=$(LINUX_DIR)/libenry.so
DARWIN_DIR=$(RESOURCES_DIR)/darwin
DARWIN_SHARED_LIB=$(DARWIN_DIR)/libenry.dylib
HEADER_FILE=libenry.h
NATIVE_LIB=./shared/enry.go

# source files to be patched for using "rubex" instead of "regexp"
RUBEX_PATCHED := internal/code-generator/generator/heuristics.go internal/tokenizer/tokenize.go common.go
RUBEX_ORIG := $(RUBEX_PATCHED:=.orig)

.PHONY: revert-oniguruma

$(LINGUIST_PATH):
	git clone https://github.com/github/linguist.git $@

clean-linguist:
	rm -rf $(LINGUIST_PATH)

clean-shared:
	rm -rf $(RESOURCES_DIR)

clean: clean-linguist clean-shared

code-generate: $(LINGUIST_PATH)
	mkdir -p data
	go run internal/code-generator/main.go

benchmarks: $(LINGUIST_PATH)
	go test -run=NONE -bench=. && benchmarks/linguist-total.sh

benchmarks-samples: $(LINGUIST_PATH)
	go test -run=NONE -bench=. -benchtime=5us && benchmarks/linguist-samples.rb

benchmarks-slow: $(LINGUST_PATH)
	mkdir -p benchmarks/output && go test -run=NONE -bench=. -slow -benchtime=100ms -timeout=100h >benchmarks/output/enry_samples.bench && \
	benchmarks/linguist-samples.rb 5 >benchmarks/output/linguist_samples.bench

$(RUBEX_ORIG): %.orig : %
	sed -i.orig -e 's/"regexp"/regexp "github.com\/moovweb\/rubex"/g' $<
	@touch $@

oniguruma: $(RUBEX_ORIG)

revert-oniguruma:
	@for file in $(RUBEX_PATCHED); do if [ -e "$$file.orig" ]; then mv "$$file.orig" "$$file" && echo mv "$$file.orig" "$$file"; fi; done

build-cli:
	go build -o enry -ldflags "$(LOCAL_LDFLAGS)" cmd/enry/main.go

linux-shared: $(LINUX_SHARED_LIB)

darwin-shared: $(DARWIN_SHARED_LIB)

$(DARWIN_SHARED_LIB):
	mkdir -p $(DARWIN_DIR) && \
	CC="o64-clang" CXX="o64-clang++" CGO_ENABLED=1 GOOS=darwin go build -buildmode=c-shared -o $(DARWIN_SHARED_LIB) $(NATIVE_LIB) && \
	mv $(DARWIN_DIR)/$(HEADER_FILE) $(RESOURCES_DIR)/$(HEADER_FILE)

$(LINUX_SHARED_LIB):
	mkdir -p $(LINUX_DIR) && \
	GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o $(LINUX_SHARED_LIB) $(NATIVE_LIB) && \
	mv $(LINUX_DIR)/$(HEADER_FILE) $(RESOURCES_DIR)/$(HEADER_FILE)
