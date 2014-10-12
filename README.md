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
  - [Build Requirements](#build-requirements)
  - [Building](#building)
  - [Logstash Integration](#logstash-integration)
  - [Generating Certificates and Keys](#generating-certificates-and-keys)
- [Documentation](#documentation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

Log Courier implements the following features:

* Tail active log files
* Follow rotations
* Suspend tailing on no more updates
* Tail STDIN stream
* Set extra fields to values (host=name), arrays (tags=[one,two]) or hashes
(origin={host=name,IP=address})
* Secure TLS shipping transport with server certificate verification
* TLS client certificate verification
* Secure CurveZMQ shipping transport to load balance across multiple Logstash
instances (optional, requires ZeroMQ 4+)
* Plaintext TCP shipping transport for configuration simplicity in local networks
* Plaintext ZMQ shipping transport
* Reload configuration without restarting
* [Administration utility](docs/AdministrationUtility.md) to monitor the
shipping speed and status
* [Multiline](docs/codecs/Multiline.md) codec
* [Filter](docs/codecs/Filter.md) codec

Log Courier integrates with Logstash using an event receiver ruby gem. An event
sender ruby gem is also available to allow fast and secure transmission between
two Logstash instances.

## Installation

### Build Requirements

1. The [golang](http://golang.org/doc/install) compiler tools (1.2 or 1.3)
1. [Logstash](http://logstash.net) 1.4.x
1. (Optional) [ZeroMQ](http://zeromq.org/intro:get-the-software) (>=3.2 for
plaintext ZMQ, >=4.0 for secure CurveZMQ)

### Building

To build without the optional ZMQ support, simply run `make` as
follows.

	git clone https://github.com/driskell/log-courier
	cd log-courier
	make

The log-courier program can then be found in the 'bin' folder.

To build with the optional ZMQ support use the following.

	git clone https://github.com/driskell/log-courier
	cd log-courier
	make with=zmq3

For CurveZMQ support (ZMQ with public key encryption) replace `zmq3` with
`zmq4`.

*If you receive errors whilst running `make` try `gmake` instead.*

### Logstash Integration

Log Courier does not utilise the lumberjack Logstash plugin and instead uses its
own custom plugin. This allows significant enhancements to the integration far
beyond the lumberjack protocol allows.

Details instructions on the plugin and how to install it into Logstash can be
found on the [Logstash Integration](docs/LogstashIntegration.md) page.

### Generating Certificates and Keys

Running `make selfsigned` will automatically build and run the `lc-tlscert`
utility that can quickly and easily generate a self-signed certificate for the
TLS shipping transport.

Likewise, running `make curvekey` will automatically build and run the
`lc-curvekey` utility that can quickly and easily generate CurveZMQ key pairs
for the CurveZMQ shipping transport.

Both tools also generate the required configuration file snippets.

## Documentation

* [Administration Utility](docs/AdministrationUtility.md)
* [Command Line Arguments](docs/CommandLineArguments.md)
* [Configuration](docs/Configuration.md)
* [Change Log](docs/ChangeLog.md)
