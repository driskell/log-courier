# Log Courier Suite

[![Golang](https://github.com/driskell/log-courier/actions/workflows/go.yml/badge.svg)](https://github.com/driskell/log-courier/actions/workflows/go.yml)
[![Ruby](https://github.com/driskell/log-courier/actions/workflows/ruby-test.yml/badge.svg)](https://github.com/driskell/log-courier/actions/workflows/ruby-test.yml)
[![Release](https://img.shields.io/github/release/driskell/log-courier.svg)](https://github.com/driskell/log-courier/releases/latest)

The Log Courier Suite is a set of lightweight tools created to ship and process log files speedily and securely, with low resource usage, to Apache Doris, Elasticsearch or Logstash instances.

- [Log Courier Suite](#log-courier-suite)
  - [Log Courier](#log-courier)
  - [Log Carver](#log-carver)
  - [Philosophy](#philosophy)
  - [Documentation](#documentation)
    - [Installation](#installation)
    - [Reference](#reference)
  - [Upgrading from 1.x to 2.x](#upgrading-from-1x-to-2x)

## Log Courier

Log Courier is a lightweight shipper. It reads from log files and transmits events over the Courier protocol to a remote Log Carver or Logstash instance.

- Reads from [files](docs/log-courier/Configuration.md#files) or the program [input](docs/log-courier/Configuration.md#stdin), following log file rotations and movements
- Compliments log events with [additional fields](docs/log-courier/Configuration.md#fields)
- Live [configuration reload](docs/log-courier/Configuration.md#reloading-configuration)
- Transmits securely using TLS with server and [client verification](docs/log-courier/Configuration.md#ssl-certificate)
- Codecs for client-side preprocessing of [multiline](docs/log-courier/codecs/Multiline.md) events and [filtering](docs/log-courier/codecs/Filter.md) of unwanted events
- Native JSON [reader](docs/log-courier/Configuration.md#reader) to support JSON files, even those with no line-termination that makes line-based reading problematic
- Remote [Administration Utility](docs/AdministrationUtility.md) to inspect monitored log files and connections in real time.
- Compatible with all supported versions of Logstash. At the time of writing this is `>= 7.7.x`.

## Log Carver

Log Carver is a lightweight event processor similar to Logstash. It receives events over the Courier protocol and performs actions against them to manipulate them into the required format for storage within Apache Doris, Elasticsearch, or further processing in Logstash. Connected clients do not receive acknowledgements until the events are acknowledged by the endpoint, whether that be Apache Doris, Elasticsearch or another more centralised Log Carver, providing end-to-end guarantees.

- Receives events securely using TLS with [client verification](docs/log-carver/Configuration.md#ssl-client-ca-receiver)
- Supports If/ElseIf/Else [conditionals](docs/log-carver/Configuration.md#conditionals) to process different events in different ways
- Provides several powerful actions for [date processing](docs/log-carver/actions/Date.md), [grokking](docs/log-carver/actions/Grok.md), or simply [computing a new field](docs/log-carver/actions/SetField.md)
- Support for complex [expressions](docs/log-carver/Configuration.md#expression) when setting fields or performing conditionals
- Transmit events for storage using the [elasticsearch transport](docs/log-carver/Configuration.md#transport) immediately after processing
- Remote [Administration Utility](docs/AdministrationUtility.md) to inspect connections in real time.

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

- [Log Courier Configuration](docs/log-courier/Configuration.md)
- [Log Carver Configuration](docs/log-carver/Configuration.md)

- [Administration Utility](docs/AdministrationUtility.md)
- [Change Log](CHANGELOG.md)
- [Command Line Arguments](docs/CommandLineArguments.md)
- [Event Format](docs/Events.md)
- [Logstash Integration](docs/LogstashIntegration.md)
- [Protocol Specification](docs/Protocol.md)
- [SSL Certificate Utility](docs/SSLCertificateUtility.md)

## Upgrading from 1.x to 2.x

There are many breaking changes in the configuration between 1.x and 2.x. Please
check carefully the list of breaking changes here:
[Change Log](CHANGELOG.md#200).

Packages also now default to using a `log-courier` user. If you require the old
behaviour of `root`, please be sure to modify the `/etc/sysconfig/log-courier`
(CentOS/RedHat) or `/etc/default/log-courier` (Ubuntu) file.
