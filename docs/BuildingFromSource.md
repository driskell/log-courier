# Building from Source

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Building from Source](#building-from-source)
  - [Requirements](#requirements)
  - [Linux, Unix, OS X](#linux-unix-os-x)
    - [Setting up a Go workspace](#setting-up-a-go-workspace)
    - [Building Log Courier](#building-log-courier)
    - [Results](#results)
  - [Windows](#windows)
    - [Setting up a Go workspace](#setting-up-a-go-workspace-1)
    - [Building Log Courier](#building-log-courier-1)
    - [Results](#results-1)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Requirements

1. Linux, Unix, OS X or Windows
1. The [Golang](http://golang.org/doc/install) compiler tools (1.13+ recommended)

## Linux, Unix, OS X

### Setting up a Go workspace

First you will need to setup a Go workspace. If you already have one setup, you
can skip this step.

Replace `~/Golang` with any path you'd like to you use. The path should not
already exist.

```
export GOPATH=~/Golang
mkdir -p "$GOPATH"
```

Also, ensure that the Go binaries are available on the command line by running
`go version`. If this doesn't work, check your Go installation is correct.

### Building Log Courier

Run the following commands to download and build the latest version of Log
Courier.

```
go get -d github.com/driskell/log-courier
cd $GOPATH/src/github.com/driskell/log-courier
go generate ./...
go install ./...
```

To build a downloaded copy of Log Courier, such as a beta version, use the
following instructions instead.

```
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

### Setting up a Go workspace

First you will need to setup a Go workspace. If you already have one setup, you
can skip this step.

Replace `C:\Golang` with any path you'd like to you use. The path should not
already exist.

```
set GOPATH=C:\Golang
mkdir %GOPATH%
```

Also, ensure that the Go binaries are available on the command line by running
`go version`. If this doesn't work, check your Go installation is correct.

### Building Log Courier

Run the following commands to download and build the latest version of Log
Courier.

```
go get -d github.com/driskell/log-courier
cd %GOPATH%/src/github.com/driskell/log-courier
go generate ./...
go install ./...
```

### Results

The log-courier binaries (log-courier, lc-tlscert, lc-admin etc.) can then be
found in the Go workspace's bin folder (e.g. `C:\Golang\bin`).
