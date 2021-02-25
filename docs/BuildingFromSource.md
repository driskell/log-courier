# Building from Source

- [Building from Source](#building-from-source)
  - [Requirements](#requirements)
  - [Building](#building)

## Requirements

1. Linux, Unix, OS X or Windows
1. The [Golang](http://golang.org/doc/install) compiler tools (1.13+ recommended)

## Building

Checkout the repository and then run the following to build the latest versions of all binaries. They will be installed to your Golang binary path, which usually has a default of `~/go/bin`. You can define the `$GOBIN` environment variable to change this.

```shell
go generate ./...
go install ./...
```
