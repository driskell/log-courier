# Log Courier [![Build Status](https://travis-ci.org/driskell/log-courier.svg?branch=develop)](https://travis-ci.org/driskell/log-courier)

Log Courier is a tool created to ship log files speedily and securely to
remote [Logstash](http://logstash.net) instances for processing whilst using
small amounts of local resources. The project is an enhanced fork of
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder) 0.3.1
with many fixes and behavioural improvements.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](http://doctoc.herokuapp.com/)*

- [Features](#features)
- [Installation](#installation)
  - [Requirements](#requirements)
  - [Building](#building)
  - [Logstash Integration](#logstash-integration)
  - [Building with ZMQ support](#building-with-zmq-support)
  - [Generating Certificates and Keys](#generating-certificates-and-keys)
- [Documentation](#documentation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

Log Courier implements the following features:

* Follow active log files
* Follow rotations
* Follow standard input stream
* Suspend tailing after periods of inactivity
* Set [extra fields](docs/Configuration.md#fields), supporting hashes and arrays
(`tags: ['one','two']`)
* [Reload configuration](docs/Configuration.md#reloading) without restarting
* Secure TLS shipping transport with server certificate verification
* TLS client certificate verification
* Secure CurveZMQ shipping transport to load balance across multiple Logstash
instances (optional, requires ZeroMQ 4+)
* Plaintext TCP shipping transport for configuration simplicity in local
networks
* Plaintext ZMQ shipping transport
* [Administration utility](docs/AdministrationUtility.md) to monitor the
shipping speed and status
* [Multiline](docs/codecs/Multiline.md) codec
* [Filter](docs/codecs/Filter.md) codec
* [Logstash Integration](docs/LogstashIntegration.md) with an input and output
plugin

## Installation

### Requirements

1. \*nix, OS X or Windows
1. The [golang](http://golang.org/doc/install) compiler tools (1.2 or 1.3)
1. [git](http://git-scm.com)
1. GNU make

*\*nix: Most requirements can usually be installed by your favourite package
manager.*

*OS X: Git and GNU make are provided automatically by XCode.*

*Windows: GNU make for Windows can be found
[here](http://gnuwin32.sourceforge.net/packages/make.htm).*

### Building

To build without the optional ZMQ support, simply run `make` as
follows.

	git clone https://github.com/driskell/log-courier
	cd log-courier
	make

The log-courier program can then be found in the 'bin' folder.

*If you receive errors whilst running `make` try `gmake` instead.*

### Logstash Integration

Log Courier does not utilise the lumberjack Logstash plugin and instead uses its
own custom plugin. This allows significant enhancements to the integration far
beyond the lumberjack protocol allows.

Install using the Logstash 1.5+ Plugin manager.

	cd /path/to/logstash
	bin/logstash plugin install logstash-input-log-courier

Detailed instructions, including integration with Logstash 1.4.x, can be found
on the [Logstash Integration](docs/LogstashIntegration.md) page.

### Building with ZMQ support

To use the 'zmq' and 'plainzmq' transports, you will need to install
[ZeroMQ](http://zeromq.org/intro:get-the-software) (>=3.2 for cleartext
plainzmq, >=4.0 for encrypted zmq).

*\*nix: ZeroMQ >=3.2 is usually available via the package manager. ZeroMQ >=4.0
may need to be built and installed manually.*

*OS X: ZeroMQ can be installed via [Homebrew](http://brew.sh).*

*Windows: ZeroMQ will need to be built and installed manually.*

Once the required version of ZeroMQ is installed, run the corresponding `make`
command to build Log Courier with the ZMQ transports.

	# ZeroMQ >=3.2 - cleartext 'plainzmq' transport
	make with=zmq3
	# ZeroMQ >=4.0 - both cleartext 'plainzmq' and encrypted 'zmq' transport
	make with=zmq4

**Please ensure that the versions of ZeroMQ installed on the Logstash hosts and
the Log Courier hosts are of the same major version. A Log Courier host that has
ZeroMQ 4.0.5 will not work with a Logstash host using ZeroMQ 3.2.4 (but will
work with a Logstash host using ZeroMQ 4.0.4.)**

*If you receive errors whilst running `make` try `gmake` instead.*

### Generating Certificates and Keys

Running `make selfsigned` will automatically build and run the `lc-tlscert`
utility that can quickly and easily generate a self-signed certificate for the
TLS shipping transport.

Likewise, running `make curvekey` will automatically build and run the
`lc-curvekey` utility that can quickly and easily generate CurveZMQ key pairs
for the CurveZMQ shipping transport. This tool is only available when Log
Courier is built with ZeroMQ >=4.0.

Both tools also generate the required configuration file snippets.

## Documentation

* [Administration Utility](docs/AdministrationUtility.md)
* [Command Line Arguments](docs/CommandLineArguments.md)
* [Configuration](docs/Configuration.md)
* [Change Log](docs/ChangeLog.md)
