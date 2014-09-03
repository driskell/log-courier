# Administration Utility

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](http://doctoc.herokuapp.com/)*

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

In order to connect, the `admin enabled` general configuration entry must be set
to `true` and an `admin port` specified. See [Configuration](Configuration.md)
for more information on these options.

## Available Commands

### `help`

Displays a list of available commands.

### `reload`

Requests Log Courier to reload its configuration.

### `status`

Displays Log Courier's current shipping status in YAML format for ease of
parsing by external scripts and utilities.

Following is an example of the output this command provides.

	/var/log/syslog (0xc21004fa20):
	  Status: Running
	  Harvester:
	    Speed (Lps): 1914.93
	    Speed (Bps): 152600.51
	    Processed lines: 5000
	    Last offset: 398893
	    Last EOF: 398893
	/var/log/maillog (0xc21004fc60):
	  Status: Running
	  Harvester:
	    Speed (Lps): 2742.58
	    Speed (Bps): 218799.19
	    Processed lines: 5000
	    Last offset: 398893
	    Last EOF: 398893

## Command Line Options

The `lc-admin` command accepts the following command line options.

	-connect="tcp:127.0.0.1:1234": the Log Courier host to connect to (tcp:ip:port or unix:path)
	-quiet=false: quietly execute the command line argument and output only the result
	-version=false: display the Log Courier client version
	-watch=false: repeat the command specified on the command line every second
