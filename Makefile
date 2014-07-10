.PHONY: go-check all test clean selfsigned

export GOPATH := ${PWD}

TAGS :=
BINS := bin/log-courier bin/lc-tlscert
TESTS := spec/courier_spec.rb spec/tcp_spec.rb spec/gem_spec.rb spec/multiline_spec.rb

ifeq ($(with),zmq3)
TAGS := $(TAGS) zmq zmq_3_x
TESTS := $(TESTS) spec/plainzmq_spec.rb
endif
ifeq ($(with),zmq4)
TAGS := $(TAGS) zmq zmq_4_x
BINS := $(BINS) bin/lc-curvekey
TESTS := $(TESTS) spec/plainzmq_spec.rb spec/zmq_spec.rb
endif

ifneq ($(implyclean),yes)
LASTTAGS := $(shell cat .Makefile.tags 2>/dev/null)
ifneq ($(LASTTAGS),$(TAGS))
IMPLYCLEAN := $(shell $(MAKE) implyclean=yes clean)
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
	@go version | grep -q 'go version go1.[123]' || (echo "Go version 1.1.x, 1.2.x or 1.3.x required, you have a version of go that is not supported."; false)

clean:
	go clean -i ./...
ifneq ($(implyclean),yes)
	rm -rf vendor/bundle
endif

.SECONDEXPANSION:
bin/%: $$(wildcard src/%/*.go) | go-check
	go get -d -tags "$(TAGS)" $*
	go install -tags "$(TAGS)" $*
