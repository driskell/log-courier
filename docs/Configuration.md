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
  - [`add host field`](#add-host-field)
  - [`add offset field`](#add-offset-field)
  - [`add path field`](#add-path-field)
  - [`add timezone field`](#add-timezone-field)
  - [`codecs`](#codecs)
  - [`dead time`](#dead-time)
  - [`fields`](#fields)
- [`admin`](#admin)
  - [`admin enabled`](#admin-enabled)
  - [`admin listen address`](#admin-listen-address)
- [`files`](#files)
  - [`paths`](#paths)
- [`general`](#general)
  - [`log file`](#log-file)
  - [`global fields`](#global-fields)
  - [`host`](#host)
  - [`log level`](#log-level)
  - [`log stdout`](#log-stdout)
  - [`log syslog`](#log-syslog)
  - [`line buffer bytes`](#line-buffer-bytes)
  - [`max line bytes`](#max-line-bytes)
  - [`persist directory`](#persist-directory)
  - [`prospect interval`](#prospect-interval)
  - [`spool max bytes`](#spool-max-bytes)
  - [`spool size`](#spool-size)
  - [`spool timeout`](#spool-timeout)
- [`includes`](#includes)
- [`network`](#network)
  - [`failure backoff`](#failure-backoff)
  - [`failure backoff max`](#failure-backoff-max)
  - [`max pending payloads`](#max-pending-payloads)
  - [`method`](#method)
  - [`reconnect backoff`](#reconnect-backoff)
  - [`reconnect backoff max`](#reconnect-backoff-max)
  - [`rfc 2782 srv`](#rfc-2782-srv)
  - [`rfc 2782 service`](#rfc-2782-service)
  - [`servers`](#servers)
  - [`ssl ca`](#ssl-ca)
  - [`ssl certificate`](#ssl-certificate)
  - [`ssl key`](#ssl-key)
  - [`timeout`](#timeout)
  - [`transport`](#transport)
- [`stdin`](#stdin)

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
harvested log file is closed you can adjust the [`dead time`](#dead-time)
option.

In the case of a network configuration change, Log Courier will disconnect and
reconnect as required at the earliest opportunity.

*Configuration reload is not currently available on Windows builds of Log
Courier.*

## Examples

Several configuration examples are available for you perusal.

* [Ship a single log file](examples/example-single.conf)
* [Ship a folder of log files](examples/example-folder.conf)
* [Ship from STDIN](examples/example-stdin.conf)
* [Ship logs with extra field information](examples/example-fields.conf)
* [Multiline log processing](examples/example-multiline.conf)

The configuration is documented in full below.

## Field Types

### String, Number, Boolean, Array, Dictionary

These are JSON types and follow the same rules. Strings within double quotes,
arrays of fields within square brackets separated by commas, and dictionaries
of key value pairs within curly braces and each entry, in the form `key":
value`, separated by a comma.

### Duration

This can be either a number or a string describing the duration. A number will
always be interpreted in seconds.

* `5` = 5 seconds
* `300` = 5 minutes (which is 300 seconds)
* `5s` = 5 seconds
* `15m` = 15 minutes

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

* `/var/log/*.log`
* `/var/log/program/log_????.log`
* `/var/log/httpd/access.log`
* `/var/log/httpd/access.log.[0-9]`

## Stream Configuration

Stream Configuration parameters can be specified for file groups within
[`files`](#files) and also for [`stdin`](#stdin). They customise the log
entries produced by passing, for example, by passing them through a codec and
adding extra fields.

### `add host field`

*Boolean. Optional. Default: true*

Adds an automatic "host" field to generated events that contains the `host`
value from the general configuration section.

### `add offset field`

*Boolean. Optional. Default: true*

Adds an automatic "offset" field to generated events that contains the current
offset in the current data stream.

*Beware that this value will reset when a file rotates or is truncated and is
generally not useful. It will be kept configurable to allow full compatibility
with Logstash Forwarder's traditional behaviour, and from version 2 the default
will be changed to false.*

### `add path field`

*Boolean. Optional. Default: true*

Adds an automatic "path" field to generated events that contains the path to the
current data stream. For stdin, this field is set to a hyphen, "-".

### `add timezone field`

*Boolean. Optional. Default: false*

Adds an automatic "timezone" field to generated events that contains the local
machine's local timezone in the format, "-0700 MST".

### `codecs`

*Codec configuration. Optional. Default: `[ { "name": "plain" } ]`*  
*Configuration reload will only affect new or resumed files*

*Depending on how log-courier was built, some codecs may not be available. Run
`log-courier -list-supported` to see the list of codecs available in a specific
build of log-courier.*

The specified codecs will receive the lines read from the log stream and perform
any decoding necessary to generate events. The plain codec does nothing and
simply ships the events unchanged.

When multiple codecs are specified, the first codec will receive events, and the
second codec will receive the output from the first codec. This allows versatile
configurations, such as combining events into multiline events and then
filtering those that aren't required.

All configurations are an array of dictionaries with at least a "name" key.
Additional options can be provided if the specified codec allows.

* `[ { "name": "codec-name" } ]`
* `[ { "name": "codec-name", "option1": "value", "option2": "42" } ]`
* `[ { "name": "first-name" }, { "name": "second-name" } ]`

Aside from "plain", the following codecs are available at this time.

* [Filter](codecs/Filter.md)
* [Multiline](codecs/Multiline.md)

### `dead time`

*Duration. Optional. Default: "1h"*  
*Configuration reload will only affect new or resumed files*

If a log file has not been modified in this time period, it will be closed and
Log Courier will simply watch it for modifications. If the file is modified it
will be reopened.

If a log file that is being harvested is deleted, it will remain on disk until
Log Courier closes it. Therefore it is important to keep this value sensible to
ensure old log files are not kept open preventing deletion.

### `fields`

*Dictionary. Optional*  
*Configuration reload will only affect new or resumed files*

Extra fields to attach to events prior to shipping. These can be simple strings,
numbers or even arrays and dictionaries.

Examples:

* `{ "type": "syslog" }`
* `{ "type": "apache", "server_names": [ "example.com", "www.example.com" ] }`
* `{ "type": "program", "program": { "exec": "program.py", "args": [ "--run", "--daemon" ] } }`

## `admin`

The admin configuration enables or disabled the REST interface within Log
Courier and allows you to configure the interface to listen on.

### `admin enabled`

*Boolean. Optional. Default: false*  
*Requires restart*

Enables the REST interface. The `lc-admin` utility can be used to connect to
this.

### `admin listen address`

*String. Optional. Default: tcp:127.0.0.1:1234*

The address the RESET interface should listen on in the format
`transport:address`.

Allowed transports are "tcp", "tcp4", "tcp6" (Windows and *nix) and "unix"
(*nix only). For the tcp transports the address format is `host:port`. For the
unix transport the address should specify a filename to use when creating the
unix domain socket. If no transport name is specified, "tcp" is assumed.

Examples:

    127.0.0.1:1234
    tcp:127.0.0.1:1234
    unix:/var/run/log-courier/admin.socket

## `files`

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
also have [Stream Configuration](#stream-configuration) parameters specified.

### `paths`

*Array of Fileglobs. Required*

At least one Fileglob must be specified and all matching files for all provided
globs will be monitored.

If the log file is rotated, Log Courier will detect this and automatically start
harvesting the new file. It will also keep the old file open to catch any
delayed writes that a still-reloading application has not yet written. You can
configure the time period before this old log file is closed using the
[`dead time`](#dead-time) option.

See above for a description of the Fileglob field type.

*To read from stdin, see the [`-stdin`](CommandLineArguments.md#stdin) command
line argument.*

Examples:

* `[ "/var/log/*.log" ]`
* `[ "/var/log/program/log_????.log" ]`
* `[ "/var/log/httpd/access.log", "/var/log/httpd/access.log.[0-9]" ]`

## `general`

The general configuration affects the general behaviour of Log Courier, such
as where to store its persistence data or how often to scan for the appearence
of new log files.

### `log file`

*Filepath. Optional*  
*Requires restart*

A log file to save Log Courier's internal log into. May be used in conjunction with `log stdout` and `log syslog`.

### `global fields`

*Dictionary. Optional*
*Configuration reload will only affect new or resumed files*

Extra fields to attach to events prior to shipping. This is identical in
behaviour to the `fields` Stream Configuration and applies globally to the
`stdin` section and to all files listed in the `files` section.

### `host`

*String. Optional. Default: System FQDN.*  
*Configuration reload will only affect new or resumed files*

Every event has an automatic field, "host", that contains the current system
FQDN. Using this option allows a custom value to be given to the "host" field
instead of the system FQDN.

### `log level`

*String. Optional. Default: "info".  
Available values: "critical", "error", "warning", "notice", "info", "debug"*  
*Requires restart*

The minimum level of detail to produce in Log Courier's internal log.

### `log stdout`

*Boolean. Optional. Default: true*  
*Requires restart*

Enables sending of Log Courier's internal log to the console (stdout). May be used in conjunction with `log syslog` and `log file`.

### `log syslog`

*Boolean. Optional. Default: false*  
*Requires restart*

Enables sending of Log Courier's internal log to syslog. May be used in conjunction with `log stdout` and `log file`.

*This option is ignored by Windows builds.*

### `line buffer bytes`

*Number. Optional. Default: 16384*

The size of the line buffer used when reading files.

If `max line bytes` is greater than this value, any lines that exceed this size
will trigger additional memory allocations. This value should be set to a value
just above the 90th percentile (or average) line length.

### `max line bytes`

*Number. Optional. Default: 1048576*

The maxmimum line length to process. If a line exceeds this length, it will be
split across multiple events. Each split line will have a "tag" field added
containing the tag "splitline". The final part of the line will not have a "tag"
field added.

If the `fields` configuration already contained a "tags" entry, and it is an
array, it will be appended to. Otherwise, the "tag" field will be left as is.

This setting can not be greater than the `spool max bytes` setting.

### `persist directory`

*String. Required*  
*Requires restart*

The directory that Log Courier should store its persistence data in.

At the time of writing, the only file saved here is `.log-courier` that
contains the offset in the file that Log Courier needs to resume from after a
graceful restart or crash. The offset is only updated when the remote endpoint
acknowledges receipt of the events.

### `prospect interval`

*Duration. Optional. Default: 10*

How often Log Courier should check for changes on the filesystem, such as the
appearance of new log files, rotations and deletions.

### `spool max bytes`

*Number. Optional. Default: 10485760*

The maximum size of an event spool, before compression. If an incomplete spool
does not have enough room for the next event, it will be flushed immediately.

If this value is modified, the receiving end should also be configured with the
new limit. For the Logstash plugin, this is the `max_packet_size` setting.

The maximum value for this setting is 2147483648 (2 GiB).

### `spool size`

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

### `spool timeout`

*Duration. Optional. Default: 5*

The maximum amount of time to wait for a full spool. If an incomplete spool is
not filled within this time limit, the spool will be flushed immediately.

## `includes`

*Array of Fileglobs. Optional*

Includes should be an array of additional file group configuration files to
read. Each configuration file should follow the format of the `files` section.

    "includes": [ "/etc/log-courier/conf.d/*.conf" ]

A file at `/etc/log-courier/conf.d/apache.conf` could then contain the
following.

    [ {
        "paths": [ "/var/log/httpd/access.log" ],
        "fields": { "type": "access_log" }
    } ]

## `network`

The network configuration tells Log Courier where to ship the logs, and also
what transport and security to use.

### `failure backoff`

*Duration. Optional. Default: 0*

Pause this long before allowing a failed endpoint to be used again. On each
consecutive failure, the pause is exponentially increased.

When set to 0, there is no backoff after the first failure. The second failure
then pauses for 1 second and begins to exponentially increase on each
consecutive failure.

(This is distinct from `reconnect backoff` in that it handles all failures, and
is available for all transports. For example, if a remote endpoint keeps failing
almost immediately after a successful connection, this backoff would prevent it
from being used again immediately and slowing down the entire pipeline.)

### `failure backoff max`

*Duration. Optional. Default: 300s*

The maximum time to wait before using a failed endpoint again. This prevents the
exponential increase of `failure backoff` from becoming too high.

### `max pending payloads`

*Number. Optional. Default: 4*

The maximum number of spools that can be in transit to a single endpoint at any
one time. Each spool will be kept in memory until the remote endpoint
acknowledges it.

If Log Courier has sent this many spools to a remote endpoint, and has not yet
received acknowledgement responses for them (either because the remote endpoint
is busy or because the link has high latency), it will pause and wait before
sending anymore.

*For most installations you should leave this at the default as it is high
enough to maintain throughput even on high latency links and low enough not to
cause excessive memory usage.*

### `method`

*String. Optional. Default: "random"
Available values: "random", "failover", "loadbalance"*

Specified the method to use when managing multiple `servers`.

`random`: Connect to a random endpoint on startup. If the connected endpoint
fails, close that connection and reconnect to another random endpoint. Protection
is added such that during reconnection, a different endpoint to the one that just
failed is guaranteed. This is the same behaviour as Log Courier 1.x.

`failover`: The endpoint list acts as a preference list with the first endpoint
in the list the preferred endpoint, and every one after that the next preferred
endpoints in the order given. The preferred endpoint is connected to initially,
and only when that connection fails is another less preferred endpoint connected
to. Log Courier will continually attempt to connect to more preferred endpoints
in the background, failing back to the most preferred endpoint if one becomes
available and closing any connections to less preferred endpoints.

`loadbalance`: Connect to all endpoints and load balance events between them.
Faster endpoints will receive more events than slower endpoints. The strategy
for load balancing is dynamic based on the acknowledgement latency of the
available endpoints.

### `reconnect backoff`

*Duration. Optional. Default: 0  
Available when `transport` is one of: `tcp`, `tls`*

Pause this long before reconnecting to a endpoint. If the remote endpoint is
completely down, this slows down the rate of reconnection attempts. On each
consecutive failure, the pause is exponentially increased.

When set to 0, the initial reconnect attempt is made immediately. The second
attempt then pauses for 1 second and begins to exponentially increase on each
consecutive failure.

### `reconnect backoff max`

*Duration. Optional. Default: 300s  
Available when `transport` is one of: `tcp`, `tls`*

The maximum time to wait between reconnect attempts. This prevents the
exponential increase of `reconnect backoff` from becoming too high.

### `rfc 2782 srv`

*Boolean. Optional. Default: true*

When performing SRV DNS lookups for entries in the [`servers`](#servers) list,
use RFC 2782 style lookups of the form `_service._proto.example.com`.

### `rfc 2782 service`

*String. Optional. Default: "courier"*

Specifies the service to request when using RFC 2782 style SRV lookups. Using
the default, "courier", an "@example.com" endpoint entry would result in a
lookup for `_courier._tcp.example.com`.

### `servers`

*Array of Strings. Required*

Sets the list of endpoints to send logs to. Accepted formats for each endpoint
entry are:

* `ipaddress:port`
* `hostname:port` (A DNS lookup is performed)
* `@hostname` (A SRV DNS lookup is performed, with further DNS lookups if
required)

How multiple endpoints are managed is defined by the `method` configuration.

### `ssl ca`

*Filepath. Required  
Available when `transport` is one of: `tls`*

Path to a PEM encoded certificate file to use to verify the connected endpoint.

### `ssl certificate`

*Filepath. Optional  
Available when `transport` is one of: `tls`*

Path to a PEM encoded certificate file to use as the client certificate.

### `ssl key`

*Filepath. Required with `ssl certificate`  
Available when `transport` is one of: `tls`*

Path to a PEM encoded private key to use with the client certificate.

### `timeout`

*Duration. Optional. Default: 15*

This is the maximum time Log Courier will wait for a endpoint to respond to a
request after logs were send to it. If the endpoint does not respond within this
time period the connection will be closed and reset.

### `transport`

*String. Optional. Default: "tls"  
Available values: "tcp", "tls"*

<!-- *Depending on how log-courier was built, some transports may not be available.
Run `log-courier -list-supported` to see the list of transports available in
a specific build of log-courier.* -->

Sets the transport to use when sending logs to the endpoints. "tls" is
recommended for most users.

"tcp" is an **insecure** equivalent to "tls" that does not encrypt traffic or
authenticate the identity of endpoints. This should only be used on trusted
internal networks. If in doubt, use the secure authenticating transport "tls".

## `stdin`

The stdin configuration contains the
[Stream Configuration](#stream-configuration) parameters that should be used
when Log Courier is set to read log data from stdin using the
[`-stdin`](CommandLineArguments.md#stdin) command line entry.
