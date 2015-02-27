# Log Courier [![Build Status](https://travis-ci.org/driskell/log-courier.svg?branch=develop)](https://travis-ci.org/driskell/log-courier)

Log Courier is a tool created to ship log files speedily and securely to
remote [Logstash](http://logstash.net) instances for processing whilst using
small amounts of local resources. The project is an enhanced fork of
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder) 0.3.1
with many fixes and behavioural improvements.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Features](#features)
- [Installation](#installation)
- [Public Repositories](#public-repositories)
  - [RPM](#rpm)
  - [DEB](#deb)
- [Building From Source](#building-from-source)
- [Logstash Integration](#logstash-integration)
- [ZeroMQ support](#zeromq-support)
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

## Public Repositories

### RPM

The author maintains a **COPR** repository with RedHat/CentOS compatible RPMs
that may be installed using `yum`. This repository depends on the widely used
**EPEL** repository for dependencies.

The **EPEL** repository can be installed automatically on CentOS distributions
by running `yum install epel-release`. Otherwise, you may follow the
instructions on the [EPEL homepage](https://fedoraproject.org/wiki/EPEL).

To install the Log Courier repository, download the corresponding `.repo`
configuration file below, and place it in `/etc/yum.repos.d`. Log Courier may
then be installed using `yum install log-courier`.

* **CentOS/RedHat 6.x**: [driskell-log-courier-epel-6.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-6/driskell-log-courier-epel-6.repo) 
* **CentOS/RedHat 7.x**:
[driskell-log-courier-epel-7.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-6/driskell-log-courier-epel-7.repo)

***NOTE:*** *The RPM packages versions of Log Courier are built using ZeroMQ 3.2 and
therefore do not support the encrypted `zmq` transport. They do support the
unencrypted `plainzmq` transport.*

### DEB

A Debian/Ubuntu compatible **PPA** repository is under consideration. At the moment,
no such repository exists.

## Building From Source

You will need the following:

1. Linux, Unix, OS X or Windows
1. The [golang](http://golang.org/doc/install) compiler tools (1.2-1.4)
1. [git](http://git-scm.com)
1. GNU make

***Linux/Unix:*** *Most requirements can usually be installed by your favourite package
manager.*  
***OS X:*** *Git and GNU make are provided automatically by XCode.*  
***Windows:*** *GNU make for Windows can be found
[here](http://gnuwin32.sourceforge.net/packages/make.htm).*

To build the binaries, simply run `make` as follows.

	git clone https://github.com/driskell/log-courier
	cd log-courier
	make

The log-courier program can then be found in the 'bin' folder. This can be
manually installed anywhere on your system. Startup scripts for various
platforms can be found in the [contrib/initscripts](contrib/initscripts) folder.

*Note: If you receive errors whilst running `make`, try `gmake` instead.*

## Logstash Integration

Log Courier does not utilise the lumberjack Logstash plugin and instead uses its
own custom plugin. This allows significant enhancements to the integration far
beyond the lumberjack protocol allows.

Install using the Logstash 1.5+ Plugin manager.

	cd /path/to/logstash
	bin/logstash plugin install logstash-input-log-courier

Detailed instructions, including integration with Logstash 1.4.x, can be found
on the [Logstash Integration](docs/LogstashIntegration.md) page.

## ZeroMQ support

To use the 'plainzmq' or 'zmq' transports, you will need to install
[ZeroMQ](http://zeromq.org/intro:get-the-software) (>=3.2 for 'plainzmq', >=4.0
for 'zmq' which supports encryption).

***Linux\Unix:*** *ZeroMQ >=3.2 is usually available via the package manager.
ZeroMQ >=4.0 may need to be built and installed manually.*  
***OS X:*** *ZeroMQ can be installed via [Homebrew](http://brew.sh).*  
***Windows:*** *ZeroMQ will need to be built and installed manually.*

Once the required version of ZeroMQ is installed, run the corresponding `make`
command to build Log Courier with the ZMQ transports.

	# ZeroMQ >=3.2 - cleartext 'plainzmq' transport
	make with=zmq3
	# ZeroMQ >=4.0 - both cleartext 'plainzmq' and encrypted 'zmq' transport
	make with=zmq4

*Note: If you receive errors whilst running `make`, try `gmake` instead.*

**Please ensure that the versions of ZeroMQ installed on the Logstash hosts and
the Log Courier hosts are of the same major version. A Log Courier host that has
ZeroMQ 4.0.5 will not work with a Logstash host using ZeroMQ 3.2.4 (but will
work with a Logstash host using ZeroMQ 4.0.4.)**

## Generating Certificates and Keys

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
* [Logstash Integration](docs/LogstashIntegration.md)
* [Change Log](docs/ChangeLog.md)
