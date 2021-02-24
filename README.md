# Log Courier Suite

[![Build Status](https://img.shields.io/github/workflow/status/driskell/log-courier/CI.svg?label=CI)](https://travis-ci.org/driskell/log-courier)
[![Latest Release](https://img.shields.io/github/release/driskell/log-courier.svg)](https://github.com/driskell/log-courier/releases/latest)

The Log Courier Suite is a set of lightweight tools created to ship and process
log files speedily and securely, with low resource usage, to Elasticsearch or
Logstash instances.

- [Log Courier Suite](#log-courier-suite)
  - [Log Courier](#log-courier)
    - [Compatible Logstash Versions](#compatible-logstash-versions)
  - [Log Carver](#log-carver)
  - [Philosophy](#philosophy)
  - [Documentation](#documentation)
    - [Installation](#installation)
    - [Reference](#reference)
  - [Upgrading from 1.x to 2.x](#upgrading-from-1x-to-2x)

## Log Courier

Log Courier is a lightweight shipper. It reads from log files and transmits events over the Courier protocol to a remote Logstash or Log Carver instance.

- Reads from files or the program input ([stdin](docs/log-courier/Configuration.md#stdin))
- Follows log file rotations and movements
- Compliments log events with [extra fields](docs/log-courier/Configuration.md#fields)
- [Reloads configuration](docs/log-courier/Configuration.md#reloading-configuration) without restarting
- Transmits securely using TLS with server and (optionally) client verification
- Monitors shipping speed and status which can be read using the [Administration utility](docs/AdministrationUtility.md)
- Pre-processes events on the sending side using codecs (e.g. [Multiline](docs/log-courier/codecs/Multiline.md), [Filter](docs/log-courier/codecs/Filter.md))
- Ships JSON files without line-terminations using a custom JSON [reader](docs/log-courier/Configuration.md#reader)

### Compatible Logstash Versions

Log Courier is compatible with most Logstash versions with a single exception. `>=7.4.0` and `<7.6.0` use a version of JRuby that has a bug making it incompatible and causes log-courier events to stop processing after an indeterminable amount of time (see #370) - please upgrade to 7.6.0 which updates JRuby to a compatible version.

## Log Carver

Log Carver is a lightweight event processor. It receives events over the Courier
protocol and performs actions against them to manipulate them into the required
format for storage within Elasticsearch, or further processing in Logstash.

- Receives events securely using TLS with optional client verification
- Supports Common Expression Language (CEL) conditional expressions in If/ElseIf/Else target different actions against different events
- Provides several actions: date, geoip, user_agent, kv, add_tag, remove_tag, set_field, unset_field
- The set_field action supports Common Expression Language (CEL) for type conversions and string building
- Transmits events to Elasticsearch using the bulk API

## Philosophy

- Keep resource usage low and predictable at all times
- Be efficient, reliable and scalable
- At-least-once delivery of events, a crash should never lose events
- Offer secure transports
- Be easy to use

## Documentation

### Installation

- [Public Repositories](docs/PublicRepositories.md)
- [Building from Source](docs/BuildingFromSource.md)

### Reference

- [Administration Utility](docs/AdministrationUtility.md)
- [Command Line Arguments](docs/CommandLineArguments.md)
- [Log Courier Configuration](docs/log-courier/Configuration.md)
- [Log Carver Configuration](docs/log-carver/Configuration.md)
- [Logstash Integration](docs/LogstashIntegration.md)
- [SSL Certificate Utility](docs/SSLCertificateUtility.md)
- [Change Log](CHANGELOG.md)

## Upgrading from 1.x to 2.x

There are many breaking changes in the configuration between 1.x and 2.x. Please
check carefully the list of breaking changes here:
[Change Log](CHANGELOG.md#200).

Packages also now default to using a `log-courier` user. If you require the old
behaviour of `root`, please be sure to modify the `/etc/sysconfig/log-courier`
(CentOS/RedHat) or `/etc/default/log-courier` (Ubuntu) file.
