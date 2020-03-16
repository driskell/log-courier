# Building from Source

- [Building from Source](#building-from-source)
  - [Requirements](#requirements)
  - [Linux, Unix, OS X](#linux-unix-os-x)
    - [Setting up a Go workspace](#setting-up-a-go-workspace)
    - [Building Log Courier](#building-log-courier)
    - [Results](#results)
  - [Windows](#windows)
    - [Setting up a Go workspace on Windows](#setting-up-a-go-workspace-on-windows)
    - [Building Log Courier on Windows](#building-log-courier-on-windows)
    - [Results on Windows](#results-on-windows)

## Requirements

1. Linux, Unix, OS X or Windows
1. The [Golang](http://golang.org/doc/install) compiler tools (1.13+ recommended)

## Linux, Unix, OS X

### Setting up a Go workspace

First you will need to setup a Go workspace. If you already have one setup, you
can skip this step.

Replace `~/Golang` with any path you'd like to you use. The path should not
already exist.

```shell
export GOPATH=~/Golang
mkdir -p "$GOPATH"
```

Also, ensure that the Go binaries are available on the command line by running
`go version`. If this doesn't work, check your Go installation is correct.

### Building Log Courier

Run the following commands to download and build the latest version of Log
Courier.

```shell
go get -d github.com/driskell/log-courier
cd $GOPATH/src/github.com/driskell/log-courier
go generate ./...
go install ./...
```

To build a downloaded copy of Log Courier, such as a beta version, use the
following instructions instead.

```shell
mkdir -p $GOPATH/src/github.com/driskell/log-courier
*Place the contents of the downloaded copy into the above folder*
go generate ./...
go install ./...
```

### Results

The log-courier binaries (log-courier, lc-tlscert, lc-admin etc.) can then be
found in the Go workspace's bin folder (e.g. `~/Golang/bin`).

Some ready-made service scripts for various platforms can be found in the
[contrib/initscripts](contrib/initscripts) folder of the Log Courier repository.

## Windows

*WARNING: These instructions have not yet been tested. If you have any problems
please create a new issue. If you needed to do something different, please raise
a pull request to update this with what works! Thanks.*

### Setting up a Go workspace on Windows

First you will need to setup a Go workspace. If you already have one setup, you
can skip this step.

Replace `C:\Golang` with any path you'd like to you use. The path should not
already exist.

```shell
set GOPATH=C:\Golang
mkdir %GOPATH%
```

Also, ensure that the Go binaries are available on the command line by running
`go version`. If this doesn't work, check your Go installation is correct.

### Building Log Courier on Windows

Run the following commands to download and build the latest version of Log
Courier.

```shell
go get -d github.com/driskell/log-courier
cd %GOPATH%/src/github.com/driskell/log-courier
go generate ./...
go install ./...
```

### Results on Windows

The log-courier binaries (log-courier, lc-tlscert, lc-admin etc.) can then be
found in the Go workspace's bin folder (e.g. `C:\Golang\bin`).
