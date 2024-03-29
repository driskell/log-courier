# Log Courier Configuration

- [Log Courier Configuration](#log-courier-configuration)
  - [Overview](#overview)
  - [Location](#location)
  - [JSON Comments](#json-comments)
  - [Services](#services)
  - [Reloading Configuration](#reloading-configuration)
  - [Examples](#examples)
  - [Field Types](#field-types)
    - [String, Number, Boolean, Array, Dictionary](#string-number-boolean-array-dictionary)
    - [Duration](#duration)
    - [Fileglob](#fileglob)
  - [`admin`](#admin)
    - [`enabled`](#enabled)
    - [`listen address`](#listen-address)
  - [`files`](#files)
    - [`paths`](#paths)
  - [`general`](#general)
    - [`global fields`](#global-fields)
    - [`host`](#host)
    - [`line buffer bytes`](#line-buffer-bytes)
    - [`log file`](#log-file)
    - [`log level`](#log-level)
    - [`log stdout`](#log-stdout)
    - [`log syslog`](#log-syslog)
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
    - [`max tls version`](#max-tls-version)
    - [`method`](#method)
    - [`min tls version`](#min-tls-version)
    - [`reconnect backoff`](#reconnect-backoff)
    - [`reconnect backoff max`](#reconnect-backoff-max)
    - [`rfc 2782 service`](#rfc-2782-service)
    - [`rfc 2782 srv`](#rfc-2782-srv)
    - [`servers`](#servers)
    - [`ssl ca`](#ssl-ca)
    - [`ssl certificate`](#ssl-certificate)
    - [`ssl key`](#ssl-key)
    - [`timeout`](#timeout)
    - [`transport`](#transport)
  - [`stdin`](#stdin)
  - [Stream Configuration](#stream-configuration)
    - [`add host field`](#add-host-field)
    - [`add offset field`](#add-offset-field)
    - [`add path field`](#add-path-field)
    - [`add timezone field`](#add-timezone-field)
    - [`add timezone name field`](#add-timezone-name-field)
    - [`codecs`](#codecs)
    - [`dead time`](#dead-time)
    - [`enable ecs`](#enable-ecs)
    - [`fields`](#fields)
    - [`hold time`](#hold-time)
    - [`reader`](#reader)
    - [`reader` limits](#reader-limits)

## Overview

The Log Courier configuration file is in YAML format. It can also be in JSON
format is desired.

When loading a configuration file, the format is determined by the file
extension. If the file extension is not recognised, Log Courier reports an
error.

- `.yaml` and `.yml` - YAML
- `.json` and `.conf` - JSON

## Location

The service configuration for Log Courier in the packaged versions (using the service wrappers from the [`contrib`](contrib) folder) will load the configuration from `/etc/log-courier/log-courier.yaml`. When running Log Courier manually a configuration file can be specified using the [`-config`](CommandLineArguments.md#-configpath) argument.

## JSON Comments

It is generally preferred to use YAML as comments are natively supported. However, the JSON format used does support comments.

End-of-line comments start with a pound sign outside of a string, and cause all characters until the end of the line to be ignored. Block comments start with a forwarder slash and an asterisk and cause all characters, including new lines, to be ignored until an asterisk followed by a forwarder slash is encountered.

```text
{
    "section": {
        # This is a comment
    }, # (these are end-of-line comments)
    "list": [
        /* More configuration here
        (this is a block comment) */
    ]
}
```

## Services

Packaged versions of Log Courier include a service wrapper so you can use `systemctl` or `service` to start, stop and reload the configuration. The default user is `log-courier` in these versions and can be change in the service configuration file. This file resides at either `/etc/sysconfig/log-courier` or `/etc/default/log-courier` depending on the package.

## Reloading Configuration

When running log-courier manually, configuration reload can be done by sending it a SIGHUP signal. To send this signal, run the following command replacing 1234 with the Process ID of Log Courier.

```shell
kill -HUP 1234
```

Log Courier will reopen its own log file if one has been configured, allowing native log rotation to take place.

Please note that files Log Courier has already started harvesting will continue to be harvested after the reload with their original configuration; the reload process will only affect new files. Additionally, harvested log files will not be reopened. Log rotations are detected automatically. To control when a harvested log file is closed you can adjust the [`dead time`](#dead-time) option.

In the case of a network configuration change, Log Courier will disconnect and reconnect at the earliest opportunity.

*Configuration reload is not currently available on Windows builds of Log Courier.*

## Examples

A basic example configuration that will ship a single log files is included below. It enables the admin interface so that `lc-admin` can connect using the default address. It has a single file set configured, which reads from a single file at `/var/log/httpd/access.log`. Each line read from the file will have a `type` field added next to the `message` field that contains the line, and this `type` field will contain the string `apache`. Each event will then be transmitted using the Log Courier protocol to a single endpoint on the same machine on port 5043. This could be Logstash or Log Carver. It will use TLS to connect and verify the endpoint using the certificate at `/etc/log-courier/logstash.cer`.

```yaml
admin:
  enabled: true
files:
  - paths:
      - /var/log/httpd/access.log
    fields:
      type: apache
network:
  servers:
    - localhost:5043
  ssl ca: /etc/log-courier/logstash.cer
```

Several more configuration examples are available for perusal in the
[examples folder](examples), currently without notes. Examples use the preferred format of YAML.

- [Ship a single log file](examples/example-single.yaml)
- [Ship a folder of log files](examples/example-folder.yaml)
- [Ship from STDIN](examples/example-stdin.yaml)
- [Ship logs with extra field information](examples/example-fields.yaml)
- [Multiline log processing](examples/example-multiline.yaml)
- [Multiple codecs](examples/example-multiple-codecs.yaml)
- [JSON file shipping](examples/example-json-shipping.yaml)

An example [JSON configuration](examples/example-json.conf) is also available
that follows a single log file.

The configuration format is documented in full below.

## Field Types

### String, Number, Boolean, Array, Dictionary

These are JSON types and follow the same rules. Strings within double quotes,
arrays of fields within square brackets separated by commas, and dictionaries
of key value pairs within curly braces and each entry, in the form `key":
value`, separated by a comma.

### Duration

This can be either a number or a string describing the duration. A number will
always be interpreted in seconds.

- `5` = 5 seconds
- `300` = 5 minutes (which is 300 seconds)
- `5s` = 5 seconds
- `15m` = 15 minutes

### Fileglob

A fileglob is a string representing a file pattern.

The pattern format used is detailed at
<http://golang.org/pkg/path/filepath/#Match> and is shown below for reference:

```text
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

In addition to the above, Log Courier also supports `/**/` to match arbitrary number of folders, through use of the [doublestar package](https://pkg.go.dev/github.com/bmatcuk/doublestar). Note that matched such as `/path**/` are not valid and will be instead interpreted as `/path*/`.

- `/var/log/*.log`
- `/var/log/program/log_????.log`
- `/var/log/httpd/access.log`
- `/var/log/httpd/access.log.[0-9]`
- `/var/log/nginx/**/*.log`

## `admin`

The admin configuration enables or disabled the REST interface within Log
Courier and allows you to configure the interface to listen on.

### `enabled`

Boolean. Optional. Default: false  
Requires restart

Enables the REST interface. The `lc-admin` utility can be used to connect to
this.

### `listen address`

String. Required when `enabled` is true. Default: "tcp:127.0.0.1:1234"  
RPM/DEB Package Default: unix:/var/run/log-courier/admin.socket

The address the REST interface should listen on must be in the format
`transport:address`.

Allowed transports are "tcp", "tcp4", "tcp6" (Windows and *nix) and "unix"
(*nix only). For the tcp transports the address format is `host:port`. For the
unix transport the address should specify a filename to use when creating the
unix domain socket. If no transport name is specified, "tcp" is assumed.

Examples:

```text
127.0.0.1:1234
tcp:127.0.0.1:1234
unix:/var/run/log-courier/admin.socket
```

## `files`

The files configuration lists the file sets that contain the logs you wish to ship. It is an array of file set configurations. In addition to the configuration parameters specified below, each file group may also have [Stream Configuration](#stream-configuration) parameters specified.

If a file matches multiple entries in the configuration, it will use only the first that it encounters. Any configuration that appears after the first matching configuration will be ignored for that file.

For example:

```yaml
files:
# First file set
- paths:
  - /var/log/nginx/access*.log
  - /var/log/nginx/sites/*/access*.log
  add host field: true
  fields:
    type: nginx
# Second file group
- paths:
  - /var/log/messages
  fields:
    type: syslog
```

### `paths`

Array of Fileglobs. Required

At least one Fileglob must be specified and all matching files for all provided
globs will be monitored.

If IO errors are encountered whilst scanning for files, the first IO error will be logged
as a warning by Log Courier and then subsequent scans will ignore IO errors to ensure that
the logs are not flooded. Each time the configuration is reloaded, IO error detection will
be reset, logging only the first IO error again.

If the log file is rotated, Log Courier will detect this and automatically start
harvesting the new file. It will also keep the old file open to catch any
delayed writes that a still-reloading application has not yet written. You can
configure the time period before this old log file is closed using the
[`dead time`](#dead-time) option.

See above for a description of the Fileglob field type.

*To read from stdin, see the [`-stdin`](CommandLineArguments.md#stdin) command line argument.*

Examples:

- `/var/log/*.log`
- `/var/log/program/log_????.log`
- `/var/log/httpd/access.log.[0-9]`

## `general`

The general configuration affects the general behaviour of Log Courier, such
as where to store its persistence data or how often to scan for the appearence
of new log files.

### `global fields`

Dictionary. Optional  
Configuration reload will only affect new or resumed files

Extra fields to attach to events prior to shipping. This is identical in
behaviour to the [`fields`](#fields) Stream Configuration and applies globally to the
`stdin` section and to all files listed in the `files` section.

### `host`

String. Optional. Default: System FQDN.  
Configuration reload will only affect new or resumed files

Every event has an automatic field, "host", that contains the current system
FQDN. Using this option allows a custom value to be given to the "host" field
instead of the system FQDN.

### `line buffer bytes`

Number. Optional. Default: 16384

See [`reader` limits](#reader-limits) for details.

### `log file`

Filepath. Optional  
Requires restart

A log file to save Log Courier's internal log into. May be used in conjunction with `log stdout` and `log syslog`.

### `log level`

String. Optional. Default: "info".  
Available values: "critical", "error", "warning", "notice", "info", "debug"  
Requires restart

The minimum level of detail to produce in Log Courier's internal log.

### `log stdout`

Boolean. Optional. Default: true  
Requires restart

Enables sending of Log Courier's internal log to the console (stdout). May be used in conjunction with `log syslog` and `log file`.

### `log syslog`

Boolean. Optional. Default: false  
Requires restart

Enables sending of Log Courier's internal log to syslog. May be used in conjunction with `log stdout` and `log file`.

*This option is ignored by Windows builds.*

### `max line bytes`

Number. Optional. Default: 1048576

See [`reader` limits](#reader-limits) for details.

This setting can not be greater than the `spool max bytes` setting.

### `persist directory`

String. Required  
RPM/DEB Package Default: /var/lib/log-courier
Requires restart

The directory that Log Courier should store its persistence data in.

At the time of writing, the only file saved here is `.log-courier` that
contains the offset in the file that Log Courier needs to resume from after a
graceful restart or crash. The offset is only updated when the remote endpoint
acknowledges receipt of the events.

### `prospect interval`

Duration. Optional. Default: 10

How often Log Courier should check for changes on the filesystem, such as the
appearance of new log files, rotations and deletions.

### `spool max bytes`

Number. Optional. Default: 10485760

The maximum size of an event spool, before compression. If an incomplete spool
does not have enough room for the next event, it will be flushed immediately.

If this value is modified, the receiving end should also be configured with the
new limit. For the Logstash plugin, this is the `max_packet_size` setting.

The maximum value for this setting is 2147483648 (2 GiB).

### `spool size`

Number. Optional. Default: 1024

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

Duration. Optional. Default: 5

The maximum amount of time to wait for a full spool. If an incomplete spool is
not filled within this time limit, the spool will be flushed immediately.

## `includes`

Array of Fileglobs. Optional

Includes should be an array of additional file group configuration files to read. Each configuration file should follow the format of the `files` section.

```yaml
includes:
- /etc/log-courier/conf.d/*.yaml
```

A file at `/etc/log-courier/conf.d/apache.yaml` could then contain the following.

```yaml
- paths:
  - /var/log/httpd/access.log
  fields:
    type: access_log
```

## `network`

The network configuration tells Log Courier where to ship the logs, and also
what transport and security to use.

### `failure backoff`

Duration. Optional. Default: 0

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

Duration. Optional. Default: 300s

The maximum time to wait before using a failed endpoint again. This prevents the
exponential increase of `failure backoff` from becoming too high.

### `max pending payloads`

Number. Optional. Default: 10

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

### `max tls version`

String. Optional. Default: ""
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls`

If specified, limits the TLS version to the given value. When not specified, the TLS version is only limited by the versions supported by Golang at build time. At the time of writing, this was 1.3.

### `method`

String. Optional. Default: "random"
Available values: "random", "failover", "loadbalance"

Specified the method to use when managing multiple `servers`.

`random`: Connect to a random endpoint on startup. If the connected endpoint
fails, close that connection and reconnect to another random endpoint. Protection
is added such that during reconnection, a different endpoint to the one that just
failed is guaranteed.

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

### `min tls version`

String. Optional. Default: 1.2
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls`

Sets the minimum TLS version allowed for connections on this transport. The TLS handshake will fail for any connection that is unable to negotiate a minimum of this version of TLS.

### `reconnect backoff`

Duration. Optional. Default: 0  
Available when `transport` is one of: `tcp`, `tls`

Pause this long before reconnecting to a endpoint. If the remote endpoint is
completely down, this slows down the rate of reconnection attempts. On each
consecutive failure, the pause is exponentially increased.

When set to 0, the initial reconnect attempt is made immediately. The second
attempt then pauses for 1 second and begins to exponentially increase on each
consecutive failure.

### `reconnect backoff max`

Duration. Optional. Default: 300s  
Available when `transport` is one of: `tcp`, `tls`

The maximum time to wait between reconnect attempts. This prevents the
exponential increase of `reconnect backoff` from becoming too high.

### `rfc 2782 service`

String. Optional. Default: "courier"

Specifies the service to request when using RFC 2782 style SRV lookups. Using
the default, "courier", an "@example.com" endpoint entry would result in a
lookup for `_courier._tcp.example.com`.

### `rfc 2782 srv`

Boolean. Optional. Default: true

When performing SRV DNS lookups for entries in the [`servers`](#servers) list,
use RFC 2782 style lookups of the form `_service._proto.example.com`.

### `servers`

Array of Strings. Required

Sets the list of endpoints to send logs to. Accepted formats for each endpoint
entry are:

- `ipaddress:port`
- `hostname:port` (A DNS lookup is performed)
- `@hostname` (A SRV DNS lookup is performed, with further DNS lookups if
required)

How multiple endpoints are managed is defined by the `method` configuration.

### `ssl ca`

Filepath. Required when `transport` is `tls`

Path to a PEM encoded certificate file to use to verify the connected endpoint.

NOTE: SHA1 signed certificates will be no longer supported for security reasons
from version 2.10.0. Setting the environment variable `GODEBUG` to `x509sha1=1`
will temporarily enable support, but users should update their certificates.

### `ssl certificate`

Filepath. Optional  
Available when `transport` is `tls`

Path to a PEM encoded certificate file to use as the client certificate. If specified, [`ssl key`](#ssl-key) is also required.

NOTE: SHA1 signed certificates will be no longer supported for security reasons
from version 2.10.0. Setting the environment variable `GODEBUG` to `x509sha1=1`
will temporarily enable support, but users should update their certificates.

### `ssl key`

Filepath. Optional  
Available when `transport` is `tls`

Path to a PEM encoded private key to use with the client certificate. If specified, [`ssl certificate`](#ssl-certificate) is also required.

### `timeout`

Duration. Optional. Default: 15

This is the maximum time Log Courier will wait for a endpoint to respond to a
request after logs were send to it. If the endpoint does not respond within this
time period the connection will be closed and reset.

### `transport`

String. Optional. Default: "tls"  
Available values: "tcp", "tls"

*Depending on how log-courier was built, some transports may not be available. Run `log-courier -list-supported` to see the list of transports available in a specific build of log-courier.*

Sets the transport to use when sending logs to the endpoints. "tls" is recommended for most users.

"tls" sends events to a host using the Courier protocol, such as Log Carver, or a Logstash instance with the `logstash-input-courier` plugin. The [`ssl ca`](#ssl-ca) option is required. You can also specify a client certificate if the remote side requires client certificate authentication by specifying [`ssl certificate`](#ssl-certificate) and [`ssl key`](#ssl-key)

"tcp" is the **insecure** equivalent of "tls" but without encryption and peer verification and should only be used on internal networks. It has no required options.

## `stdin`

The stdin configuration contains the [Stream Configuration](#stream-configuration) parameters that should be used when Log Courier is set to read log data from stdin using the [`-stdin`](CommandLineArguments.md#stdin) command line entry.

For example:

```yaml
stdin:
  add host field: true
```

## Stream Configuration

Stream Configuration parameters can be specified for file groups within
[`files`](#files) and also for [`stdin`](#stdin). They customise the log
entries produced by passing, for example, by passing them through a codec and
adding extra fields.

### `add host field`

Boolean. Optional. Default: true

Adds an automatic "host" field to generated events that contains the `host`
value from the general configuration section.

With [`enable ecs`](#enable-ecs), two fields will appear with the same value at `host.name` and `host.hostname`.
Please read the documentation for [`enable ecs`](#enable-ecs) as this field change is not backwards compatible.

### `add offset field`

Boolean. Optional. Default: true

Adds an automatic "offset" field to generated events that contains the current
offset in the current data stream.

With [`enable ecs`](#enable-ecs), this field will appear at `log.offset`.

*Beware that this value will reset when a file rotates or is truncated and is
generally not useful. It will be kept configurable to allow full compatibility
with Logstash Forwarder's traditional behaviour, and from version 2 the default
will be changed to false.*

### `add path field`

Boolean. Optional. Default: true

Adds an automatic "path" field to generated events that contains the path to the
current data stream. For stdin, this field is set to a hyphen, "-".

### `add timezone field`

Boolean. Optional. Default: false

Adds an automatic "timezone" field to generated events that contains the local
machine's local timezone in the format, "-0700 MST".

NOTE: This field CANNOT be parsed using the Logstash Date Filter plugin. Use `add timezone name field` to add a field compatible with this filter.

With [`enable ecs`](#enable-ecs), this field will appear at `event.timezone`.

### `add timezone name field`

Adds an automatic "timezone_name" field to generated events that contains the local
machine's local timezone name, such as "UTC" or "Europe/London".

This field can be parsed using the Logstash Date Filter plugin.

### `codecs`

Codec configuration. Optional. Default: Single `plain` codec  
Configuration reload will only affect new or resumed files

The specified codecs will receive the lines read from the log stream and perform
any decoding necessary to generate events. The plain codec does nothing and
simply ships the events unchanged.

When multiple codecs are specified, the first codec will receive events, and the
second codec will receive the output from the first codec. This allows versatile
configurations, such as combining events into multiline events and then
filtering those that aren't required.

All configurations are an array of dictionaries with at least a "name" key.
Additional options can be provided if the specified codec allows.

```yaml
- name: codec-name
```

```yaml
- name: codec-name
  option1: value
  option2: 42
```

```yaml
- name: first-name
- name: second-name
```

Aside from `plain`, which has no options, the following codecs are available.

- [Filter](codecs/Filter.md)
- [Multiline](codecs/Multiline.md)

*Depending on how log-courier was built, some codecs may not be available. Run `log-courier -list-supported` to see the list of codecs available in a specific build of log-courier.*

### `dead time`

Duration. Optional. Default: "1h"  
Configuration reload will only affect new or resumed files

If a log file has not been successfuly read from this time period, it will be
closed and Log Courier will simply watch it for modifications. If the file is
modified it will be reopened.

### `enable ecs`

Boolean. Optional. Default: false  

This will become default in a future major version change as a breaking change, and should always be enabled when using Log Carver as the templates installed into Elasticsearch by Log Carver require this. However, you could use Log Carver's pipeline to rewrite the events so you do not need to modify your Log Courier configurations, or so you can support both formats.

Enable Elastic Common Schema (ECS) fields in events. By default, events are generated in a similar style to Filebeat and the original Logstash Forwarder. This will enable ECS compatible fields instead **which are not backwards compatible**. This will need a change to the template used within Elasticsearch to make the fields usable. Additionally, you will need to ensure you are using fresh indexes as the ECS field types differ in such a way Elasticsearch will refuse to store them if it had previously stored non-ECS fields. Specifically, the `host` field changes from a string to an object containing `name` and `hostname`.

See [Event Format](../Events.md#event-format) for more information.

### `fields`

Dictionary. Optional  
Configuration reload will only affect new or resumed files

Extra fields to attach to events prior to shipping. These can be simple strings,
numbers or even arrays and dictionaries.

Examples:

```yaml
type: syslog
```

```yaml
type: apache
server_names:
- example.com
- www.example.com
```

```yaml
type: program
program:
  exec: program.py
  args:
  - '--run'
  - '--daemon'
```

It is worth noting that these fields will override any added via the various Stream Configuration options such as `add host field`.

There are some fields that are reserved and require special treatment, such as `@timestamp` and `tags`. See the [Events](../Events.md) documentation for information on the structure of an event and the reserved fields.

### `hold time`

Duration. Optional. Default: "96h"
Configuration reload will only affect new or resumed files
Since 2.5.5

If a log file is deleted, and this amount of time has passed and Log Courier still
has the file open, the file will be closed regardless of whether data will be lost.

This is a failsafe to ensure that a blocked pipeline does not cause deleted files
to be held open indefinitely, eventually causing the disk space to fill. This will
mean that disk usage cannot be used to detect issues sending logs and so additional
monitoring may be needed to detect this.

Set to 0 to disable and keep files open indefinitely until all data inside them is
sent and the dead_time passes.

### `reader`

String. Optional. Default: "line".  
Available Values: "line", "json".
Since 2.6.0

Specifies the reader to use for the files.

"line": This reader will emit a single event for each line.

"json": This reader will emit a single event for each JSON value in the file. This will occur even if the object does not have a new line or other whitespace following it, allowing for the reading of single json-object files with no line ending in the file. This is in contract to the "line" reader would wait for a line ending to be written. Only JSON objects are supported and the emitted event will contain all the fields and nested fields of that object.

### `reader` limits

"line": If the line exceeds the `max line bytes` configuration it will be truncated and emitted as multiple events, each of up to `max line bytes` in length. Each split line will have a "tag" field added containing the tag "splitline" to all events emitted for the line. If the `fields` configuration already contained a "tags" entry, and it is not an array, the "splitline" tag will not be added to maintain the requested value of "tags". The `line buffer bytes` is the amount of memory to allocate for each read from the file and should be sized for the median length of a line. Where a line is longer, additional memory will be allocated for that line before being released immediately. The default values are unlikely to need changing as they were chosen based on a variety of log types including syslogs, error logs and access logs.

"json": If the object's encoding exceeds `max line bytes` in length the reader will abort with an error and cease processing of the file, as it will be unable to complete reading the object within known memory bounds, and therefore unable to locate the end of the object and the start of the next. Like the "line" reader, the `line buffer bytes` pre-allocates memory for reading and should be sized to the median size of an object in its JSON encoding.
