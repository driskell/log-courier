# Log Courier Configuration

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](http://doctoc.herokuapp.com/)*

- [Overview](#overview)
- [Reloading](#reloading)
- [Examples](#examples)
- [Field Types](#field-types)
  - [String, Number, Boolean, Array, Dictionary](#string-number-boolean-array-dictionary)
  - [Duration](#duration)
  - [Fileglob](#fileglob)
- [`"general"`](#general)
  - [`"persist directory"`](#persist-directory)
  - [`"prospect interval"`](#prospect-interval)
  - [`"log level"`](#log-level)
- [`"network"`](#network)
  - [`"transport"`](#transport)
  - [`"servers"`](#servers)
  - [`"ssl ca"`](#ssl-ca)
  - [`"ssl certificate"`](#ssl-certificate)
  - [`"ssl key"`](#ssl-key)
  - [`"curve server key"`](#curve-server-key)
  - [`"curve public key"`](#curve-public-key)
  - [`"curve secret key"`](#curve-secret-key)
  - [`"timeout"`](#timeout)
  - [`"reconnect"`](#reconnect)
- [`"files"`](#files)
  - [`"paths"`](#paths)
  - [`"fields"`](#fields)
  - [`"dead time"`](#dead-time)
  - [`"codec"`](#codec)
- [`"includes"`](#includes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The Log Courier configuration is currently stored in standard JSON format with
the exception that comments are allowed.

End-of-line comments start with a pound sign outside of a string, and cause all
characters until the end of the line to be ignored. Block comments start with
a forwarder slash and an asterisk and cause all characters, including new lines,
to be ignored until an asterisk followed by a forwarder slash is encountered.

```
{
	"general": {
		# General configuration here
	},
	"network": {
		# Network configuration here
	}, # (these are end-of-line comments)
	"files": {
		/* File configuration here
		(this is a block comment) */
	}
}
```

## Reloading

Log Courier can reload its configuration without the need for a restart. It will
do this upon receiving the SIGHUP signal. To send this signal, run the following
command replacing 1234 with the Process ID of Log Courier.

	kill -HUP 1234

Please note that files Log Courier has already started harvesting will continue
to harvest after the reload until their dead time is reached. The reload process
will only affect the scanning of new files and the network configuration. In the
case of a network configuration change, Log Courier will disconnect and
reconnect at the earliest opportunity.

*Configuration reload is not currently available on Windows builds of Log
Courier.*

## Examples

Several configuration examples are available for you perusal.

* [Ship a single log file](examples/example-single.conf)
* [Ship a folder of log files](examples/example-folder.conf)
* [Ship from STDIN](examples/example-stdin.conf)
* [Ship logs with extra field information](examples/example-fields.conf)
* [Multiline log processing](examples/example-multiline.conf)
* [Using ZMQ to load balance](examples/example-zmq.conf)

The configuration is documented in full below.

## Field Types

### String, Number, Boolean, Array, Dictionary

These are JSON types and follow the same rules. Strings within double quotes,
arrays of fields within square brackets separated by commas, and dictionaries
of key value pairs within curly braces and each entry, in the form string:
field, separated by a comma.

### Duration

This can be either a number or a string describing the duration. A number will
always be interpreted in seconds.

* `5` = 5 seconds
* `300` = 5 minutes (which is 300 seconds)
* `"5s"` = 5 seconds
* `"15m"` = 15 minutes

### Fileglob

A fileglob is a string representing a file pattern.

The pattern format used is detailed at
http://golang.org/pkg/path/filepath/#Match and is shown below for reference:

```
term:
	'*'         matches any sequence of non-Separator characters
	'?'         matches any single non-Separator character
	'[' [ '^' ] { character-range } ']'
							character class (must be non-empty)
	c           matches character c (c != '*', '?', '\\', '[')
	'\\' c      matches character c

character-range:
	c           matches character c (c != '\\', '-', ']')
	'\\' c      matches character c
	lo '-' hi   matches character c for lo <= c <= hi
```

* `"/var/log/*.log"`
* `"/var/log/program/log_????.log"`
* `"/var/log/httpd/access.log"`
* `"/var/log/httpd/access.log.[0-9]"`

## `"general"`

The general configuration affects the general behaviour of Log Courier, such
as where to store its persistence data or how often to scan for the appearence
of new log files.

### `"persist directory"`

*String. Optional. Default: "."*

The directory that Log Courier should store its persistence data in. The default
is the current working directory of Log Courier which is specified using the
path, `"."`.

At the time of writing, the only file saved here is `.log-courier` that
contains the offset in the file that Log Courier needs to resume from after a
graceful restart or server crash. The offset is only updated when the remote
server acknowledges receipt of the events.

### `"prospect interval"`

*Duration. Optional. Default: 10*

How often Log Courier should check for changes on the filesystem, such as the
appearance of new log files, rotations and deletions.

### `"log level"`

*String. Optional. Default: "info"
Available values: "critical", "error", "warning", "notice", "info", "debug"*

The maximum level of detailed logs to produce.

## `"network"`

The network configuration tells Log Courier where to ship the logs, and also
what transport and security to use.

### `"transport"`

*String. Optional. Default: "tls"  
Available values: "tcp", "tls", "plainzmq", "zmq"*

*Depending on how log-courier was built, some transports may not be available.
Run `log-courer -list-supported` to see the list of transports available in
a specific build of log-courier.*

Sets the transport to use when sending logs to the servers. "tls" is recommended
for most users and connects to a single server at random, reconnecting to a
different server at random each time the connection fails. "curvezmq" connects
to all specified servers and load balances events across them.

"tcp" and "plainzmq" are **insecure** equivalents to "tls" and "zmq"
respectively that do not encrypt traffic or authenticate the identity of
servers. These should only be used on trusted internal networks. If in doubt,
use the secure authenticating transports "tls" and "zmq".

"plainzmq" is only available if Log Courier was compiled with the "with=zmq3" or
"with=zmq4" options.

"zmq" is only available if Log Courier was compiled with the "with=zmq4"
option.

### `"servers"`

*Array of strings. Required*

Sets the list of servers to send logs to. DNS names are resolved into IP
addresses each time connections are made and all available IP addresses are
used.

### `"ssl ca"`

*Filepath. Required with "transport" = "tls". Not allowed otherwise*

Path to a PEM encoded certificate file to use to verify the connected server.

### `"ssl certificate"`

*Filepath. Optional with "transport" = "tls". Not allowed otherwise*

Path to a PEM encoded certificate file to use as the client certificate.

### `"ssl key"`

*Filepath. Required with "ssl certificate". Not allowed when "transport" !=
"tls"*

Path to a PEM encoded private key to use with the client certificate.

### `"curve server key"`

*String. Required with "transport" = "zmq". Not allowed otherwise*

The Z85-encoded public key that corresponds to the server(s) secret key. Used
to verify the server(s) identity. This can be generated using the Genkey tool.

### `"curve public key"`

*String. Required with "transport" = "zmq". Not allowed otherwise*

The Z85-encoded public key for this client. This can be generated using the
Genkey tool.

### `"curve secret key"`

*String. Required with "transport" = "zmq". Not allowed otherwise*

The Z85-encoded secret key for this client. This can be generated using the
Genkey tool.

### `"timeout"`

*Duration. Optional. Default: 15*

This is the maximum time Log Courier will wait for a server to respond to a
request after logs were send to it. If the server does not respond within this
time period the connection will be closed and reset.

When using the ZMQ transport, this is the maximum amount of time Log Courier
will wait for a response to a request. If any response is not received within
this time period the corresponding request is retransmitted. If no responses are
received within this time period, the entire ZMQ stack is reset.

### `"reconnect"`

*Duration. Optional. Default: 1*

Pause this long before reconnecting. If the remote server is completely down,
this slows down the rate of reconnection attempts.

When using the ZMQ transport, this is how long to wait before restarting the ZMQ
stack when it was reset.

## `"files"`

The file configuration lists the file groups that contain the logs you wish to
ship. It is an array of file group configurations. A minimum of one file group
configuration must be specified.

```
	[
		{
			# First file group
		},
		{
			# Second file group
		}
	]
```

### `"paths"`

*Array of Fileglobs. Required*

At least one Fileglob must be specified and all matching files for all provided
globs will be tailed.

See above for a description of the Fileglob field type.

Examples:

* `[ "/var/log/*.log" ]`
* `[ "/var/log/program/log_????.log" ]`
* `[ "/var/log/httpd/access.log", "/var/log/httpd/access.log.[0-9]" ]`

### `"fields"`

*Dictionary. Optional*

Extra fields to attach the event prior to shipping. These can be simple strings,
numbers or even arrays and dictionaries.

Examples:

* `{ "type": "syslog" }`
* `{ "type": "apache", "server_names": [ "example.com", "www.example.com" ] }`
* `{ "type": "program", "program": { "exec": "program.py", "args": [ "--run", "--daemon" ] } }`

### `"dead time"`

*Duration. Optional. Default: "24h"*

If a log file has not been modified in this time period, it will be closed and
Log Courier will simply watch it for modifications. If the file is modified it
will be reopened.

If a log file that is being harvested is deleted, it will remain on disk until
Log Courier closes it. Therefore it is important to keep this value sensible to
ensure old log files are not kept open preventing deletion.

### `"codec"`

*Codec configuration. Optional. Default: `{ "name": "plain" }`*

*Depending on how log-courier was built, some codecs may not be available. Run
`log-courer -list-supported` to see the list of codecs available in a specific
build of log-courier.*

The specified codec will receive the lines read from the log stream and perform
any decoding necessary to generate events. The plain codec does nothing and
simple ships the events as-is.

All configurations are a dictionary with at least a "name" key. Additional
options can be provided if the specified codec allows.

	{ "name": "codec-name" }
	{ "name": "codec-name", "option1": "value", "option2": "42" }

A single codec is available at this time.

* [Multiline](codecs/Multiline.md)

## `"includes"`

*Array of Fileglobs. Optional*

Includes should be an array of additional file group configuration files to
read. Each configuration file should follow the format of the `"files"` section.

	"includes": [ "/etc/log-courier/conf.d/*.conf" ]

A file at `/etc/log-courier/conf.d/apache.conf` could then contain the
following.

	[ {
		"paths": [ "/var/log/httpd/access.log" ],
		"fields": [ "type": "access_log" ]
	} ]
