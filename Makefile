.PHONY: go-check all

export GOPATH := ${PWD}

TAGS :=
BINS := bin/log-courier
TESTS := spec/courier_spec.rb spec/gem_spec.rb spec/multiline_spec.rb

ifeq ($(with),zmq)
	TAGS := $(TAGS) zmq zmq_4_x
	BINS := $(BINS) bin/genkey
	TESTS := $(TESTS) spec/zmq_spec.rb
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

bin/%: src/%/*.go | go-check
	go get -d -tags "$(TAGS)" $*
	go install -tags "$(TAGS)" $*

clean:
	go clean -i ./...
	rm -rf vendor/bundle
