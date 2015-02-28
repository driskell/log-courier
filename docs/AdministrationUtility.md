# Administration Utility

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Available Commands](#available-commands)
  - [`help`](#help)
  - [`reload`](#reload)
  - [`status`](#status)
- [Command Line Options](#command-line-options)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The `lc-admin` command allows you to remotely (or locally) connect to a running
Log Courier instance to monitor and control log shipping.

To enable a Log Courier instance to receive administration connections, set the
`admin enabled` general configuration entry to `true`. To specify a custom
listen address, set the `admin listen address` entry. See
[Configuration](Configuration.md) for more information on these options and the
default listen address.

The `lc-admin` utility aims to be always backwards compatible whenever possible.
This means a newer version of `lc-admin` should be able to connect to any older
version of `log-courier`. The same is not true in reverse, and an older
`lc-admin` may be unable to connect or communicate with a newer `log-courier`.

## Available Commands

### `help`

Displays a list of available commands.

### `reload`

Requests Log Courier to reload its configuration.

### `status`

*Syntax: status [<format>]*

Displays Log Courier's current shipping status in the requested format.

<format> must be yaml or json. If not specified, the default format is yaml.

Following is an example of the output this command provides.

	"State: /var/log/nginx/access.log (0xc2080681e0)":
	  Status: Running
	  Harvester:
	    Speed (Lps): 20205.40
	    Speed (Bps): 1627565.16
	    Processed lines: 43024
	    Current offset: 3473919
	    Last EOF Offset: Never
	    Status: Alive
	Prospector:
	  Watched files: 1
	  Active states: 1
	Publisher:
	  Status: Connected
	  Speed (Lps): 8735.15
	  Published lines: 23552
	  Pending Payloads: 10
	  Timeouts: 0
	  Retransmissions: 0

## Command Line Options

The `lc-admin` command accepts the following command line options.

	-connect="tcp:127.0.0.1:1234": the Log Courier address to connect to
	-quiet=false: quietly execute the command line argument and output only the result
	-version=false: display the Log Courier client version
	-watch=false: repeat the command specified on the command line every second
