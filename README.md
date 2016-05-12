# Log Courier

[![Build Status](https://img.shields.io/travis/driskell/log-courier/master.svg)](https://travis-ci.org/driskell/log-courier)
[![Latest Release](https://img.shields.io/github/release/driskell/log-courier.svg)](https://github.com/driskell/log-courier/releases/latest)

Log Courier is a lightweight tool created to ship log files speedily and
securely, with low resource usage, to remote [Logstash](http://logstash.net)
instances.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Upgrading to 2.x](#upgrading-to-2x)
- [Features](#features)
- [Philosophy](#philosophy)
- [Documentation](#documentation)
  - [Installation](#installation)
  - [Reference](#reference)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Upgrading to 2.x

There are many breaking changes in the configuration between 1.8 and 2.x. Please
check carefully the list of breaking changes here:
[Change Log](CHANGELOG.md#200).

Packages also now default to using a `log-courier` user. If you require the old
behaviour of `root`, please be sure to modify the `/etc/sysconfig/log-courier`
(CentOS/RedHat) or `/etc/default/log-courier` (Ubuntu) file.

## Features

* [Logstash Integration](docs/LogstashIntegration.md)
* Read from a file or the program input (`stdin`)
* Follow log file rotations and movements
* Compliment log events with [extra fields](docs/Configuration.md#fields)
* [Reload configuration](docs/Configuration.md#reloading) without restarting
* Transmit securely using TLS with server and (optionally) client verification
* Monitor shipping speed and status with the
[Administration utility](docs/AdministrationUtility.md)
* Pre-process events on the sender using codecs
(e.g. [Multiline](docs/codecs/Multiline.md), [Filter](docs/codecs/Filter.md))

## Philosophy

* At-least-once delivery of events to the Logstash pipeline - a Log Courier
crash should never lose events
* Be efficient, reliable and scalable
* Keep resource usage low
* Be easy to use

## Documentation

### Installation

* [Public Repositories](docs/PublicRepositories.md)
* [Building from Source](docs/BuildingFromSource.md)

### Reference

* [Administration Utility](docs/AdministrationUtility.md)
* [Command Line Arguments](docs/CommandLineArguments.md)
* [Configuration](docs/Configuration.md)
* [Logstash Integration](docs/LogstashIntegration.md)
* [SSL Certificate Utility](docs/SSLCertificateUtility.md)
* [Change Log](CHANGELOG.md)
