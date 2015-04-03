# Log Courier Configuration

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Reloading](#reloading)
- [Examples](#examples)
- [Field Types](#field-types)
  - [String, Number, Boolean, Array, Dictionary](#string-number-boolean-array-dictionary)
  - [Duration](#duration)
  - [Fileglob](#fileglob)
- [Stream Configuration](#stream-configuration)
  - [`"codec"`](#codec)
  - [`"dead time"`](#dead-time)
  - [`"fields"`](#fields)
- [`"general"`](#general)
  - [`"admin enabled"`](#admin-enabled)
  - [`"admin listen address"`](#admin-listen-address)
  - [`"log file"`](#log-file)
  - [`"host"`](#host)
  - [`"log level"`](#log-level)
  - [`"log stdout"`](#log-stdout)
  - [`"log syslog"`](#log-syslog)
  - [`"line buffer bytes"`](#line-buffer-bytes)
  - [`"max line bytes"`](#max-line-bytes)
  - [`"persist directory"`](#persist-directory)
  - [`"prospect interval"`](#prospect-interval)
  - [`"spool max bytes"`](#spool-max-bytes)
  - [`"spool size"`](#spool-size)
  - [`"spool timeout"`](#spool-timeout)
- [`"network"`](#network)
  - [`"curve server key"`](#curve-server-key)
  - [`"curve public key"`](#curve-public-key)
  - [`"curve secret key"`](#curve-secret-key)
  - [`"max pending payloads"`](#max-pending-payloads)
  - [`"reconnect"`](#reconnect)
  - [`"rfc 2782 srv"`](#rfc-2782-srv)
  - [`"rfc 2782 service"`](#rfc-2782-service)
  - [`"servers"`](#servers)
  - [`"ssl ca"`](#ssl-ca)
  - [`"ssl certificate"`](#ssl-certificate)
  - [`"ssl key"`](#ssl-key)
  - [`"timeout"`](#timeout)
  - [`"transport"`](#transport)
- [`"files"`](#files)
  - [`"paths"`](#paths)
- [`"includes"`](#includes)
- [`"stdin"`](#stdin)

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

Log Courier will reopen its own log file if one has been configured, allowing
native log rotation to take place.

Please note that files Log Courier has already started harvesting will continue
to be harvested after the reload with their original configuration; the reload
process will only affect new files. Additionally, harvested log files will not
be reopened. Log rotations are detected automatically. To control when a
harvested log file is closed you can adjust the [`"dead time"`](#dead-time)
option.

In the case of a network configuration change, Log Courier will disconnect and
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
of key value pairs within curly braces and each entry, in the form `"key":
value`, separated by a comma.

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

## Stream Configuration

Stream Configuration parameters can be specified for file groups within
[`"files"`](#files) and also for [`"stdin"`](#stdin). They customise the log
entries produced by passing, for example, by passing them through a codec and
adding extra fields.

### `"codec"`

*Codec configuration. Optional. Default: `{ "name": "plain" }`*
*Configuration reload will only affect new or resumed files*

*Depending on how log-courier was built, some codecs may not be available. Run
`log-courier -list-supported` to see the list of codecs available in a specific
build of log-courier.*

The specified codec will receive the lines read from the log stream and perform
any decoding necessary to generate events. The plain codec does nothing and
simply ships the events unchanged.

All configurations are a dictionary with at least a "name" key. Additional
options can be provided if the specified codec allows.

{ "name": "codec-name" }
{ "name": "codec-name", "option1": "value", "option2": "42" }

Aside from "plain", the following codecs are available at this time.

* [Filter](codecs/Filter.md)
* [Multiline](codecs/Multiline.md)

### `"dead time"`

*Duration. Optional. Default: "24h"*
*Configuration reload will only affect new or resumed files*

If a log file has not been modified in this time period, it will be closed and
Log Courier will simply watch it for modifications. If the file is modified it
will be reopened.

If a log file that is being harvested is deleted, it will remain on disk until
Log Courier closes it. Therefore it is important to keep this value sensible to
ensure old log files are not kept open preventing deletion.

### `"fields"`

*Dictionary. Optional*
*Configuration reload will only affect new or resumed files*

Extra fields to attach the event prior to shipping. These can be simple strings,
numbers or even arrays and dictionaries.

Examples:

* `{ "type": "syslog" }`
* `{ "type": "apache", "server_names": [ "example.com", "www.example.com" ] }`
* `{ "type": "program", "program": { "exec": "program.py", "args": [ "--run", "--daemon" ] } }`

## `"general"`

The general configuration affects the general behaviour of Log Courier, such
as where to store its persistence data or how often to scan for the appearence
of new log files.

### `"admin enabled"`

*Boolean. Optional. Default: false*  
*Requires restart*

Enables the administration listener that the `lc-admin` utility can connect to.

### `"admin listen address"`

*String. Optional. Default: tcp:127.0.0.1:1234*

The address the administration listener should listen on in the format
`transport:address`.

Allowed transports are "tcp", "tcp4", "tcp6" (Windows and *nix) and "unix"
(*nix only). For the tcp transports the address format is `host:port`. For the
unix transport the address should specify a filename to use when creating the
unix domain socket. If no transport name is specified, "tcp" is assumed.

Examples:

    127.0.0.1:1234
    tcp:127.0.0.1:1234
    unix:/var/run/log-courier/admin.socket

### `"log file"`

*Filepath. Optional*  
*Requires restart*

A log file to save Log Courier's internal log into. May be used in conjunction with `"log stdout"` and `"log syslog"`.

### `"host"`

*String. Optional. Default: System FQDN.*
*Configuration reload will only affect new or resumed files*

Every event has an automatic field, "host", that contains the current system
FQDN. Using this option allows a custom value to be given to the "host" field
instead of the system FQDN.

### `"log level"`

*String. Optional. Default: "info".  
Available values: "critical", "error", "warning", "notice", "info", "debug"*  
*Requires restart*

The minimum level of detail to produce in Log Courier's internal log.

### `"log stdout"`

*Boolean. Optional. Default: true*  
*Requires restart*

Enables sending of Log Courier's internal log to the console (stdout). May be used in conjunction with `"log syslog"` and `"log file"`.

### `"log syslog"`

*Boolean. Optional. Default: false*  
*Requires restart*

Enables sending of Log Courier's internal log to syslog. May be used in conjunction with `"log stdout"` and `"log file"`.

*This option is ignored by Windows builds.*

### `"line buffer bytes"`

*Number. Optional. Default: 16384*

The size of the line buffer used when reading files.

If `max line bytes` is greater than this value, any lines that exceed this size
will trigger additional memory allocations. This value should be set to a value
just above the 90th percentile (or average) line length.

### `"max line bytes"`

*Number. Optional. Default: 1048576*

The maxmimum line length to process. If a line exceeds this length, it will be
split across multiple events. Each split line will have a "tag" field added
containing the tag "splitline". The final part of the line will not have a "tag"
field added.

If the `fields` configuration already contained a "tags" entry, and it is an
array, it will be appended to. Otherwise, the "tag" field will be left as is.

This setting can not be greater than the `spool max bytes` setting.

### `"persist directory"`

*String. Optional. Default: "."*  
*Requires restart*

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

### `"spool max bytes"`

*Number. Optional. Default: 10485760*

The maximum size of an event spool, before compression. If an incomplete spool
does not have enough room for the next event, it will be flushed immediately.

If this value is modified, the receiving end should also be configured with the
new limit. For the Logstash plugin, this is the `max_packet_size` setting.

The maximum value for this setting is 2147483648 (2 GiB).

### `"spool size"`

*Number. Optional. Default: 1024*

How many events to spool together and flush at once. This improves efficiency
when processing large numbers of events by submitting them for processing in
bulk.

Internal benchmarks have shown that increasing to 5120, for example, can
give around a 25% boost of events per second, at the expense of more memory
usage.

*For most installations you should leave this at the default as it can
easily cope with over 10,000 events a second and uses little memory. It is
useful only in very specific circumstances.*

### `"spool timeout"`

*Duration. Optional. Default: 5*

The maximum amount of time to wait for a full spool. If an incomplete spool is
not filled within this time limit, the spool will be flushed immediately.

## `"network"`

The network configuration tells Log Courier where to ship the logs, and also
what transport and security to use.

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

### `"max pending payloads"`

*Number. Optional. Default: 10*

The maximum number of spools that can be in transit at any one time. Each spool
will be kept in memory until the remote server acknowledges it.

If Log Courier has sent this many spools to the remote server, and has not yet
received acknowledgement responses for them (either because the remote server
is busy or because the link has high latency), it will pause and wait before
sending anymore data.

*For most installations you should leave this at the default as it is high
enough to maintain throughput even on high latency links and low enough not to
cause excessive memory usage.*

### `"reconnect"`

*Duration. Optional. Default: 1*

Pause this long before reconnecting. If the remote server is completely down,
this slows down the rate of reconnection attempts.

When using the ZMQ transport, this is how long to wait before restarting the ZMQ
stack when it was reset.

### `"rfc 2782 srv"`

*Boolean. Optional. Default: true*

When performing SRV DNS lookups for entries in the [`"servers"`](#servers) list,
use RFC 2782 style lookups of the form `_service._proto.example.com`.

### `"rfc 2782 service"`

*String. Optional. Default: "courier"*

Specifies the service to request when using RFC 2782 style SRV lookups. Using
the default, "courier", an "@example.com" server entry would result in a lookup
for `_courier._tcp.example.com`.

### `"servers"`

*Array of Strings. Required*

Sets the list of servers to send logs to. Accepted formats for each server entry
are:

* `ipaddress:port`
* `hostname:port` (A DNS lookup is performed)
* `@hostname` (A SRV DNS lookup is performed, with further DNS lookups if
required)

The initial server is randomly selected. Subsequent connection attempts are made
to the next IP address available (if the server had multiple IP addresses) or to
the next server listed in the configuration file (if all addresses for the
previous server were exausted.)

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

### `"timeout"`

*Duration. Optional. Default: 15*

This is the maximum time Log Courier will wait for a server to respond to a
request after logs were send to it. If the server does not respond within this
time period the connection will be closed and reset.

When using the ZMQ transport, this is the maximum amount of time Log Courier
will wait for a response to a request. If any response is not received within
this time period the corresponding request is retransmitted. If no responses are
received within this time period, the entire ZMQ stack is reset.

### `"transport"`

*String. Optional. Default: "tls"  
Available values: "tcp", "tls", "plainzmq", "zmq"*

*Depending on how log-courier was built, some transports may not be available.
Run `log-courier -list-supported` to see the list of transports available in
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

## `"files"`

The files configuration lists the file groups that contain the logs you wish to
ship. It is an array of file group configurations.

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

In addition to the configuration parameters specified below, each file group may
also have [Stream Configuration](#streamconfiguration) parameters specified.

### `"paths"`

*Array of Fileglobs. Required*

At least one Fileglob must be specified and all matching files for all provided
globs will be monitored.

If the log file is rotated, Log Courier will detect this and automatically start
harvesting the new file. It will also keep the old file open to catch any
delayed writes that a still-reloading application has not yet written. You can
configure the time period before this old log file is closed using the
[`"dead time"`](#dead-time) option.

See above for a description of the Fileglob field type.

*To read from stdin, see the [`-stdin`](CommandLineArguments.md#stdin) command
line argument.*

Examples:

* `[ "/var/log/*.log" ]`
* `[ "/var/log/program/log_????.log" ]`
* `[ "/var/log/httpd/access.log", "/var/log/httpd/access.log.[0-9]" ]`

## `"includes"`

*Array of Fileglobs. Optional*

Includes should be an array of additional file group configuration files to
read. Each configuration file should follow the format of the `"files"` section.

    "includes": [ "/etc/log-courier/conf.d/*.conf" ]

A file at `/etc/log-courier/conf.d/apache.conf` could then contain the
following.

    [ {
        "paths": [ "/var/log/httpd/access.log" ],
        "fields": { "type": "access_log" }
    } ]

## `"stdin"`

The stdin configuration contains the
[Stream Configuration](#streamconfiguration) parameters that should be used when
Log Courier is set to read log data from stdin using the
[`-stdin`](CommandLineArguments.md#stdin) command line entry.
