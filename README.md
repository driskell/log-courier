# Log Courier

[![Build Status](https://img.shields.io/travis/driskell/log-courier/master.svg)](https://travis-ci.org/driskell/log-courier)
[![Latest Release](https://img.shields.io/github/release/driskell/log-courier.svg)](https://github.com/driskell/log-courier/releases/latest)

Log Courier is a lightweight tool created to ship log files speedily and
securely, with low resource usage, to remote [Logstash](http://logstash.net)
instances. The project is an enhanced fork of
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder) 0.3.1
with many fixes and behavioural improvements.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Features](#features)
- [Philosophy](#philosophy)
- [Differences to Logstash Forwarder](#differences-to-logstash-forwarder)
- [Public Repositories](#public-repositories)
  - [Redhat / CentOS](#redhat--centos)
  - [Ubuntu](#ubuntu)
- [Building from Source](#building-from-source)
  - [Linux / Unix / OS X](#linux--unix--os-x)
  - [Windows](#windows)
  - [Results](#results)
- [Logstash Integration](#logstash-integration)
- [Generating Certificates](#generating-certificates)
- [Documentation](#documentation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

* [Logstash Integration](docs/LogstashIntegration.md) with an input and output
plugin
* Read events from a file or a Unix pipeline
* Follow log file rotations and movements
* Close files after inactivity, reopening on change, to keep resource usage low
* Add [extra fields](docs/Configuration.md#fields) to events prior to shipping
* [Reload configuration](docs/Configuration.md#reloading) without restarting
* Ship events securely using TLS with server (and optionally client) certificate
verification
* Ship events in plaintext using TCP
* Monitor shipping speed and status with the
[Administration utility](docs/AdministrationUtility.md)
* Pre-process events using codecs (e.g. [Multiline](docs/codecs/Multiline.md),
[Filter](docs/codecs/Filter.md))
* Ship events securely using TLS with server (and optionally client) certificate
verification
* Ship events in plaintext using TCP

## Philosophy

* Aim to guarantee at-least-once delivery of events to the Logstash pipeline - a
Log Courier crash should never lose events *[1]*
* Be efficient and reliable
* Keep resource usage low

*[1] A __Logstash__ crash or output failure will still lose some events until
Logstash itself implements delivery guarantees or persistence - see
[elastic/logstash#2609](https://github.com/elastic/logstash/issues/2609) and
[elastic/logstash#2605](https://github.com/elastic/logstash/issues/2605). Log
Courier aims to provide complete compatibility with theses features as they
develop.*

## Differences to Logstash Forwarder

Log Courier is an enhanced fork of
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder) 0.3.1
with many fixes and behavioural improvements. The primary changes are:

* The publisher protocol is rewritten to avoid many causes of "i/o timeout"
which would result in duplicate events sent to Logstash
* The prospector and registrar are heavily revamped to handle log rotations and
movements far more reliably, and to report errors cleanly
* The harvester is improved to retry if an error occurred rather than stop
* The configuration can be reloaded without restarting
* An administration tool is available which can display the shipping speed and
status of all watched log files
* Fields configurations can contain arrays and dictionaries, not just strings
* Codec support is available which allows multiline processing at the sender
side
* A TCP transport is available which removes the requirement for SSL
certificates
* There is support for client SSL certificate verification
* Peer IP address and certificate DN can be added to received events in Logstash
to distinguish events send from different instances
* Windows: Log files are not locked allowing log rotation to occur
* Windows: Log rotation is detected correctly

## Public Repositories

### Redhat / CentOS

*The Log Courier repository depends on the __EPEL__ repository which can be
installed automatically on CentOS distributions by running
`yum install epel-release`. For other distributions, please follow the
installation instructions on the
[EPEL homepage](https://fedoraproject.org/wiki/EPEL).*

To install the Log Courier YUM repository, download the corresponding `.repo`
configuration file below, and place it in `/etc/yum.repos.d`. Log Courier may
then be installed using `yum install log-courier`.

* **CentOS/RedHat 6.x**: [driskell-log-courier-epel-6.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-6/driskell-log-courier-epel-6.repo)
* **CentOS/RedHat 7.x**:
[driskell-log-courier-epel-7.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-6/driskell-log-courier-epel-7.repo)

Once installed, create a configuration file at
`/etc/log-courier/log-courier.conf` to suit your needs, then start the Log
Courier service to begin shipping.

    service log-courier start

### Ubuntu

To install the Log Courier apt-get repository, run the following commands.

    sudo add-apt-repository ppa:devel-k/log-courier
    sudo apt-get update

Log Courier may then be installed using `apt-get install log-courier`.

Once installed, create a configuration file at
`/etc/log-courier/log-courier.conf` to suit your needs, then start the Log
Courier service to begin shipping.

    service log-courier start

**NOTE:** The Ubuntu packages have had limited testing and you are welcome to give
feedback and raise feature requests or bug reports to help improve them!

## Building from Source

Requirements:

1. Linux, Unix, OS X or Windows
1. GNU make
1. git
1. The [Golang](http://golang.org/doc/install) compiler tools (1.2-1.4)

### Linux / Unix / OS X

*Most requirements are usually available via your distribution's package
manager. On OS X, Git and GNU make are provided automatically by XCode.*

Run the following commands to download and build Log Courier.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make

*Note: If you receive errors whilst running `make`, try `gmake` instead.*

### Windows

*Installing [msysGit](http://msysgit.github.io/) will provide you with Git and
GNU make, and a Unix-like environment to build within.*

Run the following commands to download and build Log Courier, changing the path
to the Golang installation if necessary (the default is `C:\Go`, which in msys
terms is `/c/Go`.)

    export GOROOT=/c/Go
    export PATH=$PATH:$GOROOT/bin
    git clone https://github.com/driskell/log-courier
    cd log-courier
    make

### Results

The log-courier program can then be found in the 'bin' folder. Service scripts
for various platforms can be found in the
[contrib/initscripts](contrib/initscripts) folder, or it can be run on the
command line:

    bin/log-courier -config /path/to/config.conf

## Logstash Integration

Log Courier communicates with Logstash via an input plugin called "courier".

You may install the plugin using the Logstash 1.5 Plugin manager. Run the
following as the user Logstash was installed with.

    cd /path/to/logstash
    bin/plugin install logstash-input-courier

Detailed instructions, including integration with Logstash 1.4.x, can be found
on the [Logstash Integration](docs/LogstashIntegration.md) page.

*Note: If you receive a Plugin Conflict error, try updating the zeromq output
plugin first using `bin/plugin update logstash-output-zeromq`*

## Generating Certificates

Log Courier provides a commands to help generate SSL certificates: `lc-tlscert`.
This utility is also bundled with the packaged versions of Log Courier, and
should be immediately available at the command-line.

When building from source, running `make selfsigned` will automatically build
and run the `lc-tlscert` utility that can quickly and easily generate a
self-signed certificate, along with the corresponding configuration snippets,
for the 'tls' transport.

## Documentation

* [Administration Utility](docs/AdministrationUtility.md)
* [Command Line Arguments](docs/CommandLineArguments.md)
* [Configuration](docs/Configuration.md)
* [Logstash Integration](docs/LogstashIntegration.md)
* [Change Log](docs/ChangeLog.md)
