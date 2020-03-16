# Administration Utility

- [Administration Utility](#administration-utility)
  - [Overview](#overview)
  - [Available Commands](#available-commands)
    - [`help`](#help)
    - [`status`](#status)
    - [`prospector [status | files [id]]`](#prospector-status--files-id)
    - [`publisher [status | endpoints [id]]`](#publisher-status--endpoints-id)
    - [`reload`](#reload)
    - [`version`](#version)
    - [`debug`](#debug)
  - [Command Line Options](#command-line-options)
    - [`-config`](#config)
    - [`-connect`](#connect)
    - [`-quiet`](#quiet)
    - [`-version`](#version-1)
    - [`-watch`](#watch)

## Overview

The `lc-admin` command allows you to remotely (or locally) interact with a
running Log Courier instance to monitor and control log shipping, using the REST
API.

To enable the Log Courier REST API, set the admin
[`enabled`](Configuration.md#enabled) configuration entry to `true`. To specify
a custom listen address, set the admin
[`listen address`](Configuration.md#listen-address) option.

*NOTE: `lc-admin` version 2.0.0 and above  cannot connect to older 1.x Log
Courier instances.*

## Available Commands

### `help`

Displays a list of available commands.

### `status`

Displays a full status snapshot of all Log Courier internals.

### `prospector [status | files [id]]`

The `prospector` command will show the current status of all watched files and
their corresponding shipping status if they are actively being shipped.

Information can be narrowed down by specifying `status` or `files` as a
parameter. Information for a specific `files` entry can be requested by
following it by the internal file ID. This file ID changes on each restart of
Log Courier.

### `publisher [status | endpoints [id]]`

Show the connectivity status with the `publisher` command. This will show the
status of each connected endpoint and a summary of the overall shipping status.

Narrow the information by specifying `status` or `endpoints` as a parameter.
Information for a specific endpoint can be requested by following it by its
name in the configuration file, or by its internal ID number.

### `reload`

Requests Log Courier to reload its configuration.

### `version`

Displays the version of the connected Log Courier instance.

### `debug`

Displays a live go routine trace of the running Log Courier instance for
debugging purposes.

## Command Line Options

The `lc-admin` command accepts the following command line options.

### `-config`

RPM/DEB Package Default: /etc/log-courier/log-courier.yaml

Load the given configuration file and connect to the `admin` `listen address`
specified inside it. Ignored if `-connect` is specified.

### `-connect`

Default: tcp:127.0.0.1:1234

Connect to the REST API using the specified address. Any `-config` option is
ignored if `-connect` is used.

The address must be in the format `transport:address`.

Allowed transports are "tcp", "tcp4", "tcp6" (Windows and *nix) and "unix"
(*nix only). For the tcp transports the address format is `host:port`. For the
unix transport the address should specify a filename to use when connecting. If
no transport name is specified, "tcp" is assumed.

Examples:

    127.0.0.1:1234
    tcp:127.0.0.1:1234
    unix:/var/run/log-courier/admin.socket

### `-quiet`

Default: false

Quietly execute the command line argument and output only the result.

### `-version`

Default: false

Display the Log Courier client version.

### `-watch`

Default: false

Repeat the command specified on the command line every second
