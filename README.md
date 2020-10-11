# Log Courier Suite

[![Go 1.13 Build Status](https://img.shields.io/github/workflow/status/driskell/log-courier/Go%201.13.svg)](https://travis-ci.org/driskell/log-courier)
[![Go 1.14 Build Status](https://img.shields.io/github/workflow/status/driskell/log-courier/Go%201.14.svg)](https://travis-ci.org/driskell/log-courier)
[![Latest Release](https://img.shields.io/github/release/driskell/log-courier.svg)](https://github.com/driskell/log-courier/releases/latest)

The Log Courier Suite is a set of lightweight tools created to ship and process
log files speedily and securely, with low resource usage, to Elasticsearch or
Logstash instances.

- [Log Courier Suite](#log-courier-suite)
  - [Log Courier](#log-courier)
  - [Log Carver](#log-carver)
  - [Philosophy](#philosophy)
  - [Documentation](#documentation)
    - [Installation](#installation)
    - [Reference](#reference)
  - [Upgrading from 1.x to 2.x](#upgrading-from-1x-to-2x)

## Log Courier

Log Courier is a lightweight shipper. It reads from log files and transmits events over
the Courier protocol to a remote Logstash or Log Carver instance.

- Reads from files or the program input (`stdin`)
- Follows log file rotations and movements
- Compliments log events with [extra fields](docs/Configuration.md#fields)
- [Reloads configuration](docs/Configuration.md#reloading) without restarting
- Transmits securely using TLS with server and (optionally) client verification
- Monitors shipping speed and status which can be read using the
[Administration utility](docs/AdministrationUtility.md)
- Pre-processes events on the sending side using codecs
(e.g. [Multiline](docs/codecs/Multiline.md), [Filter](docs/codecs/Filter.md))

## Log Carver

(Beta)

Log Carver is a lightweight event processor. It receives events over the Courier
protocol and performs actions against them to manipulate them into the required
format for storage within Elasticsearch, or further processing in Logstash.

- Receives events securely using TLS with optional client verification
- Supports Common Expression Language (CEL) conditional expressions in If/ElseIf/Else
target different actions against different events
- Provides several actions: date, geoip, user_agent, kv, add_tag, remove_tag, set_field, unset_field
- The set_field action supports Common Expression Language (CEL) for type conversions and string building
- Transmits events to Elasticsearch using the bulk API

## Philosophy

- At-least-once delivery of events, a Log Courier crash should never lose events
- Be efficient, reliable and scalable
- Keep resource usage low
- Be easy to use

## Documentation

### Installation

- [Public Repositories](docs/PublicRepositories.md)
- [Building from Source](docs/BuildingFromSource.md)

### Reference

- [Administration Utility](docs/AdministrationUtility.md)
- [Command Line Arguments](docs/CommandLineArguments.md)
- [Configuration](docs/Configuration.md)
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
