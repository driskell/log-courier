.PHONY: all test go-check clean

export GOPATH := ${PWD}

TAGS :=
BINS := bin/log-courier
TESTS := spec/courier_spec.rb spec/gem_spec.rb spec/multiline_spec.rb

ifeq ($(with),zmq)
TAGS := $(TAGS) zmq zmq_4_x
BINS := $(BINS) bin/genkey
TESTS := $(TESTS) spec/zmq_spec.rb
endif

ifneq ($(skiptags),yes)
LASTTAGS := $(shell cat .Makefile.tags 2>/dev/null)
ifneq ($(LASTTAGS),$(TAGS))
IMPLYCLEAN := $(shell $(MAKE) skiptags=yes clean)
SAVETAGS := $(shell echo "$(TAGS)" >.Makefile.tags)
endif
endif

all: $(BINS)

test: all vendor/bundle/.GemfileModT
	bundle exec rspec $(TESTS)

# Only update bundle if Gemfile changes
vendor/bundle/.GemfileModT: Gemfile
	bundle install --path vendor/bundle
	touch $@

go-check:
	@go version > /dev/null || (echo "Go not found. You need to install go: http://golang.org/doc/install"; false)
	@go version | grep -q 'go version go1.[12]' || (echo "Go version 1.1.x or 1.2.x required, you have a version of go that is not supported."; false)

clean:
	go clean -i ./...
	rm -rf vendor/bundle

.SECONDEXPANSION:
bin/%: $$(wildcard src/%/*.go) | go-check
	go get -d -tags "$(TAGS)" $*
	go install -tags "$(TAGS)" $*
