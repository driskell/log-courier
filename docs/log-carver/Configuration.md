# Log Carver Configuration

- [Log Carver Configuration](#log-carver-configuration)
  - [Overview](#overview)
  - [Location](#location)
  - [JSON Comments](#json-comments)
  - [Services](#services)
  - [Reloading Configuration](#reloading-configuration)
  - [Examples](#examples)
  - [Index Templates](#index-templates)
  - [Field Types](#field-types)
    - [String, Number, Boolean, Array, Dictionary](#string-number-boolean-array-dictionary)
    - [Pattern String](#pattern-string)
    - [Duration](#duration)
    - [Expression](#expression)
  - [`admin`](#admin)
    - [`enabled`](#enabled)
    - [`listen address`](#listen-address)
  - [`general`](#general)
    - [`debug events`](#debug-events)
    - [`log file`](#log-file)
    - [`log level`](#log-level)
    - [`log stdout`](#log-stdout)
    - [`log syslog`](#log-syslog)
    - [`processor routines`](#processor-routines)
    - [`spool max bytes`](#spool-max-bytes)
    - [`spool size`](#spool-size)
    - [`spool timeout`](#spool-timeout)
  - [`grok`](#grok)
    - [`load defaults`](#load-defaults)
    - [`pattern files`](#pattern-files)
  - [`network`](#network)
    - [`failure backoff`](#failure-backoff)
    - [`failure backoff max`](#failure-backoff-max)
    - [`index pattern`](#index-pattern)
    - [`max pending payloads`](#max-pending-payloads)
    - [`max tls version`](#max-tls-version)
    - [`method`](#method)
    - [`min tls version`](#min-tls-version)
    - [`password`](#password)
    - [`reconnect backoff`](#reconnect-backoff)
    - [`reconnect backoff max`](#reconnect-backoff-max)
    - [`retry backoff`](#retry-backoff)
    - [`retry backoff max`](#retry-backoff-max)
    - [`rfc 2782 service`](#rfc-2782-service)
    - [`rfc 2782 srv`](#rfc-2782-srv)
    - [`routines`](#routines)
    - [`servers`](#servers)
    - [`ssl ca`](#ssl-ca)
    - [`ssl certificate`](#ssl-certificate)
    - [`ssl key`](#ssl-key)
    - [`template file`](#template-file)
    - [`template patterns`](#template-patterns)
    - [`timeout`](#timeout)
    - [`transport`](#transport)
    - [`username`](#username)
  - [`pipelines`](#pipelines)
    - [Actions](#actions)
    - [Conditionals](#conditionals)
  - [`receivers`](#receivers)
    - [`enabled` (receiver)](#enabled-receiver)
    - [`listen`](#listen)
    - [`max pending payloads` (receiver)](#max-pending-payloads-receiver)
    - [`max queue size` (receiver)](#max-queue-size-receiver)
    - [`max tls version` (receiver)](#max-tls-version-receiver)
    - [`min tls version` (receiver)](#min-tls-version-receiver)
    - [`name` (receiver)](#name-receiver)
    - [`ssl certificate` (receiver)](#ssl-certificate-receiver)
    - [`ssl client ca` (receiver)](#ssl-client-ca-receiver)
    - [`ssl key` (receiver)](#ssl-key-receiver)
    - [`transport` (receiver)](#transport-receiver)
    - [`verify peers` (receiver)](#verify-peers-receiver)

## Overview

The Log Carver configuration file is in YAML format. It can also be in JSON format is desired.

When loading a configuration file, the format is determined by the file extension. If the file extension is not recognised, Log Carver reports an error.

- `.yaml` and `.yml` - YAML
- `.json` and `.conf` - JSON

## Location

The service configuration for Log Carver in the packages versions (using the service wrappers from the [`contrib`](contrib) folder) will load the configuration from `/etc/log-carver/log-carver.yaml`. When running Log Carver manually a configuration file can be specified using the [`-config`](CommandLineArguments.md#-configpath) argument.

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

Packaged versions of Log Carver include a service wrapper so you can use `systemctl` or `service` to start, stop and reload the configuration. The default user is `log-carver` in these versions and can be change in the service configuration file. This file resides at either `/etc/sysconfig/log-carver` or `/etc/default/log-carver` depending on the package.

## Reloading Configuration

When running log-carver manually, configuration reload can be done by sending it a SIGHUP signal. To send this signal, run the following command replacing 1234 with the Process ID of Log Carver.

```shell
kill -HUP 1234
```

Log Carver will reopen its own log file if one has been configured, allowing native log rotation to take place.

In the case of a network configuration change, Log Carver will disconnect and reconnect at the earliest opportunity.

*Configuration reload is not currently available on Windows builds of Log Carver.*

## Examples

A basic example configuration that will receive events using the Log Courier protocol on port 12345 is below. For each event it will perform a `grok` operation if they have a `type` field of `syslog` and then tag them with `syslogs`. Other logs with a different `type` field value will be tagged with `unknown_event`. It will bulk store the events in the local Elasticsearch cluster on port 9200, in indexes such as `test-2021.01.01` that roll-over each day.

```yaml
receivers:
- listen:
  - 0.0.0.0:12345
pipelines:
- if: >-
    event.type == "syslog"
  then:
  - name: grok
    field: message
    patterns:
    - '(?P<timestamp>%{MONTH} +%{MONTHDAY} %{TIME}) (?:<%{NONNEGINT:facility}.%{NONNEGINT:priority}> )?%{IPORHOST:logsource}+(?: %{PROG:program}(?:\[%{POSINT:pid}\])?:|) %{GREEDYDATA:message}'
  - name: add_tag
    tag: syslogs
- else:
  - name: add_tag
    tag: unknown_event
network:
  transport: es
  index pattern: >-
    test-%{+2006.01.02}
  servers:
  - 127.0.0.1:9200
```

The above could be accompanied by Log Courier instances with the configuration below, which would set the `type` field to `nginx`.

```yaml
files:
  - paths:
      - /var/log/nginx/access*.log
    fields:
      type: nginx
network:
  servers:
    - localhost:12345
  ssl ca: /etc/log-courier/log-carver.crt
```

Several more configuration examples are available for perusal in the
[examples folder](examples), currently without notes. Examples use the preferred format of YAML.

- [Using KV to parse SELinux audit logs](examples/example-audit.yaml)
- [Processing nginx access logs with Grok, UserAgent and GeoIP](examples/example-nginx.yaml)
- [Converting events to Elastic Common Schema](examples/example-schema.yaml)
- [Grokking syslogs](examples/example-syslog.yaml)
- [Handling multiple tomcat catalina log formats](examples/example-tomcat.yaml)

The configuration is documented in full below.

## Index Templates

When using the `es` transport to store events into Elasticsearch, a [built-in template](../../lc-lib/transports/es/templates.go) will be installed by Log Carver under the name of `logstash`. It is checked for its existence when Log Carver first communicates to the Elasticsearch cluster endpoint. If it already exists (which it will if you previously used Logstash or Log Carver had already stored its template) then it will do nothing and continue. Log Carver will only install the template if it does not already exist.

The template can be overridden using the [`template file`](#template-file) configuration. Or just the index patterns can be changed using [`template patterns`](#template-patterns).

It should be noted that **Log Carver's default template requies the [`enable ecs`](../log-courier/Configuration.md#enable-ecs) configuration to be enabled within Log Courier**. If you are not using this setting in your Log Courier instances, you should specify your own index template, rely on an existing Logstash one, or implement a pipeline that will correct the event format. This is important as the event formats are not compatible due to the `host` field format change from a string to an object.

An example pipeline configuration that would correct the host field is as follows:

```yaml
pipelines:
- if: has(event.host) && type(event.host) == string
  then:
  - name: set_field
    field: 'host[name]'
    value: 'event.host'
  - name: set_field
    field: 'host[hostname]'
    value: 'event.host.name'
```

## Field Types

### String, Number, Boolean, Array, Dictionary

These are JSON types and follow the same rules. Strings within double quotes,
arrays of fields within square brackets separated by commas, and dictionaries
of key value pairs within curly braces and each entry, in the form `key":
value`, separated by a comma.

### Pattern String

Various actions, as well as some configurations such as [`index pattern`](#index-pattern), allow you to access the fields of events within a string so that the value to use can be generated at run-time on a per-event basis. This is done using the `%{}` syntax as shown below and demonstrated using the example event.

Using the below example event:

```yaml
"@timestamp": 2021-05-01T01:02:03Z00:00
field: Chicago
nested:
  deeper: New York
```

- To return `Chicago`, use `%{field}`
- To return `New York`, use `%{nested[deeper]}`
- To return the "@timestamp" field and take only the year, month and day, use `%{+2006-01-02}`, which will return `2021-05-01`
- To use multiple fields inside a string use something like `I love %{nested[deeper]}, but when I visited %{field} in %{+2006} it became my home`

Some notes:

- The time format is specified by writing the reference time `Mon Jan 2 15:04:05 MST 2006` in the format you desire, such as `2006-01-02`, or `2nd January`. This works as all numerical components are distinct from each other (See: [Golang "time" package constants](https://golang.org/pkg/time/#pkg-constants))
- If the field is not found, it is treated as an empty string, and so `hello %{missing}world` using the above event as an example would product `hello world`
- An invalid time format will output the invalid time format pieces unchanged, so `February 2006` in the above example would return `February 2021`, whereas a valid time format of `January 2006` would return `May 2021`.

### Duration

This can be either a number or a string describing the duration. A number will
always be interpreted in seconds.

- `5` = 5 seconds
- `300` = 5 minutes (which is 300 seconds)
- `5s` = 5 seconds
- `15m` = 15 minutes

### Expression

Expressions are used within [conditionals](#conditionals) to perform tests and calculations. They can also be used for some actions. For example, an expression can be used for the [SetField `value` option](actions/SetField.md#value) in order to create a new value for a field.

An expression is an immutable operation so will not modify anything in the event, and is used to generate new values that may then be used by mutating actions, or tested for truthness within a conditional.

Expressions use the [Common Expression Language](https://github.com/google/cel-spec/blob/master/doc/langdef.md). Of particular use will be the [Macros](https://github.com/google/cel-spec/blob/master/doc/langdef.md#macros) (such as `has`, `map` and `filter`) and [Operators and Functions](https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions) (such as `+`, `!=`, `startsWith` and `int`)

```js
// Test if a field exists, returns a boolean true if it does
has(event.field)

// Map the tags and return a list of prefixed versions
event.tags.map(tag, "prefix_" + tag)

// Filter the tags to remove unwanted tag and return a new list of tags
event.tags.filter(tag, tag != "unwanted")

// Return a string which concatonates two fields with a space between them
event.message + " " + event.field

// Multiply an integer value
event.integer * 100

// Coerce a string field to an integer then subtract 100
int(event.integer) - 100

// Test if a field starts with something
event.message.startsWith("ERROR ")
```

## `admin`

The admin configuration enables or disabled the REST interface within Log
Carver and allows you to configure the interface to listen on.

### `enabled`

Boolean. Optional. Default: false  
Requires restart

Enables the REST interface. The `lc-admin` utility can be used to connect to
this.

### `listen address`

String. Required when `enabled` is true. Default: "tcp:127.0.0.1:1234"  
RPM/DEB Package Default: unix:/var/run/log-carver/admin.socket

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
unix:/var/run/log-carver/admin.socket
```

## `general`

The general configuration affects the general behaviour of Log Carver.

### `debug events`

Boolean. Optional. Default: false

Enables debugging of event data. Events will be output to the logs in debug level messages after processing of the event is completed, to allow inspection of pipeline results.

### `log file`

Filepath. Optional  
Requires restart

A log file to save Log Carver's internal log into. May be used in conjunction with `log stdout` and `log syslog`.

### `log level`

String. Optional. Default: "info".  
Available values: "critical", "error", "warning", "notice", "info", "debug"  
Requires restart

The minimum level of detail to produce in Log Carver's internal log.

### `log stdout`

Boolean. Optional. Default: true  
Requires restart

Enables sending of Log Carver's internal log to the console (stdout). May be used in conjunction with `log syslog` and `log file`.

### `log syslog`

Boolean. Optional. Default: false  
Requires restart

Enables sending of Log Carver's internal log to syslog. May be used in conjunction with `log stdout` and `log file`.

*This option is ignored by Windows builds.*

### `processor routines`

Number. Optional. Default: 4. Min: 1. Max: 128

The number of processor routines to start. Event spools are distributed across the processor routines for parallel processing. Increasing this may allow the usage of more CPUs and therefore faster processing of events.

### `spool max bytes`

Number. Optional. Default: 10485760

The maximum size of an event spool, before compression. If an incomplete spool
does not have enough room for the next event, it will be flushed immediately.

The maximum value for this setting is 2147483648 (2 GiB).

Log Carver will combine smaller spools received over the network into spools this size, and also split larger spools received over the network to this size. Each processor thread will process a single spool at any one moment in time. The default value here is usually best but if you have increased this value on the shipping side in log-courier you may wish to increase it on this size to prevent unnecessary splitting.

### `spool size`

Number. Optional. Default: 1024

How many events to spool together and flush at once. This improves efficiency
when processing large numbers of events by submitting them for processing in
bulk.

### `spool timeout`

Duration. Optional. Default: 5

The maximum amount of time to wait for a full spool. If an incomplete spool is
not filled within this time limit, the spool will be flushed immediately.

## `grok`

The grok configuration allows customisation of the [grok](actions/Grok.md) action defaults.

### `load defaults`

Boolean. Optional. Default: true

Log Carver comes built with a pre-defined set of named patterns based on Logstash's own patterns that can be referenced by grok action patterns. This configuration allows the defaults to be disabled so that a completely custom set can be provided instead.

### `pattern files`

Array of Strings. Optional

A list of files to be loaded that contain named patterns to be used with grok actions. Log Carver is compatible with the format of Logstash's patterns files so they can provided. Please do note that Logstash patterns files will not work unmodified due to the differences in pattern syntax. See the [grok](actions/Grok.md) documentation for more details.

The format is as follows, with a single pattern on each line, with the first word of the line being the name of the pattern and the pattern following the first space encountered.

```text
NAME ^pattern$
MATCH another-pattern
```

## `network`

The network configuration tells Log Carver where to send the logs, and also what transport and security to use.

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

### `index pattern`

Pattern String. Optional. Default: logstash-%{+2006-01-02}

Specifies the indices to bulk index events into. This is a [Pattern String](#pattern-string) so can contain references to fields within the event being indexed, so that different events can be indexed into different indices. Most commonly this would include the date for automatically rolling over of indices, as in the default of `logstash-%{+2006-01-02}`.

### `max pending payloads`

Number. Optional. Default: 10

The maximum number of spools that can be in transit to a single endpoint at any
one time. Each spool will be kept in memory until the remote endpoint
acknowledges it.

If Log Carver has sent this many spools to a remote endpoint, and has not yet
received acknowledgement responses for them (either because the remote endpoint
is busy or because the link has high latency), it will pause and wait before
sending anymore.

*For most installations you should leave this at the default as it is high
enough to maintain throughput even on high latency links and low enough not to
cause excessive memory usage.*

### `max tls version`

String. Optional. Default: ""
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is one of: `tls`, `es-https`

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
to. Log Carver will continually attempt to connect to more preferred endpoints
in the background, failing back to the most preferred endpoint if one becomes
available and closing any connections to less preferred endpoints.

`loadbalance`: Connect to all endpoints and load balance events between them.
Faster endpoints will receive more events than slower endpoints. The strategy
for load balancing is dynamic based on the acknowledgement latency of the
available endpoints.

### `min tls version`

String. Optional. Default: 1.2
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is one of: `tls`, `es-https`

Sets the minimum TLS version allowed for connections on this transport. The TLS handshake will fail for any connection that is unable to negotiate a minimum of this version of TLS.

### `password`

String. Optional. Default none
Available when `transport` is one of: `es`, `es-https`

Enables Basic authentication for the transport, using this password. Use in conjunction with [`username`](#username).

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

### `retry backoff`

Duration. Optional. Default: 0  
Available when `transport` is one of: `es`, `es-https`

Pause this long before retrying a bulk operation on Elasticsearch. If the remote endpoint is overwhelmed, this slows down the rate of bulk indexing attempts. On each consecutive failure, the pause is exponentially increased.

When set to 0, the initial retry attempt is made immediately. The second attempt then pauses for 1 second and begins to exponentially increase on each consecutive failure.

### `retry backoff max`

Duration. Optional. Default: 300s  
Available when `transport` is one of: `es`, `es-https`

The maximum time to wait between retry attempts. This prevents the exponential increase of `retry backoff` from becoming too high.

### `rfc 2782 service`

String. Optional. Default: "courier"

Specifies the service to request when using RFC 2782 style SRV lookups. Using
the default, "courier", an "@example.com" endpoint entry would result in a
lookup for `_courier._tcp.example.com`.

### `rfc 2782 srv`

Boolean. Optional. Default: true

When performing SRV DNS lookups for entries in the [`servers`](#servers) list,
use RFC 2782 style lookups of the form `_service._proto.example.com`.

### `routines`

Number. Optional. Default: 4. Min: 1. Max: 32
Available when `transport` is one of: `es`, `es-https`

The number of bulk requests to perform at any one moment in time. Increasing this will make more simultaneous requests to Elasticsearch, increasing resource usage on that side, whilst increasing the speed of indexing.

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

Filepath. Required when `transport` is one of: `tls`, `es-https`

Path to a PEM encoded certificate file to use to verify the connected endpoint.

### `ssl certificate`

Filepath. Optional  
Available when `transport` is `tls`

Path to a PEM encoded certificate file to use as the client certificate. If specified, [`ssl key`](#ssl-key) is also required.

### `ssl key`

Filepath. Optional
Available when `transport` is `tls`

Path to a PEM encoded private key to use with the client certificate. If specified, [`ssl certificate`](#ssl-certificate) is also required.

### `template file`

String. Optional
Available when `transport` is one of: `es`, `es-https`

Specify a path to an index template to be installed instead of the builtin one that Log Carver provides. See [Index Templates](#index-templates) for more information.

### `template patterns`

Array of Strings. Optional. Default: Single item of "logstash-*"
Available when `transport` is one of: `es`, `es-https`

When using the builtin index template that Log Carver provides, this can be used to override the indices patterns that the template will apply to. See [Index Templates](#index-templates) for more information.

Ignored if `template file` is specified, as you can specify the patterns inside the custom template.

### `timeout`

Duration. Optional. Default: 15

This is the maximum time Log Carver will wait for a endpoint to respond to a
request after logs were send to it. If the endpoint does not respond within this
time period the connection will be closed and reset.

### `transport`

String. Optional. Default: "tls"  
Available values: "tcp", "tls", "es", "es-https"

*Depending on how log-carver was built, some transports may not be available. Run `log-carver -list-supported` to see the list of transports available in a specific build of log-carver.*

Sets the transport to use when sending logs to the endpoints. "es-https" is recommended for most users.

"es-https" sends events to an Elasticsearch cluster using HTTPS. "es" sends events using HTTP only.
It will also install a template called `logstash` if one does not exist. If you are migrating from Logstash this template will already exist. It should be compatible if you haven't used the Log Courier `enable ecs` configuration.

"tls" sends events to a host using the Courier protocol, such as Log Carver. "tcp" is the equivalent but
without TLS encryption and peer verification and should only be used on internal networks.

### `username`

String. Optional. Default none
Available when `transport` is one of: `es`, `es-https`

Enables Basic authentication for the transport, using this username. Use in conjunction with [`password`](#password).

## `pipelines`

Array of Actions. Optional. Default none

This is a list of actions to perform against every event passing through Log Carver. Parallelism is controlled by the [`processor routines`](#processor-routines) configuration and the results of the pipeline can be inspected in the logs when [`debug events`](#debug-events) is enabled.

There are two types of entries that can appear in a pipeline. Actions, and Conditionals.

### Actions

Actions have a single consistent field, `name`, that denotes which action to take. Each action has a unique set of fields that configure its behaviour.

For example:

```yaml
- name: action
  option: value
  option2: 42
```

Available actions are:

- [Add Tag](actions/AddTag.md)
- [Date](actions/Date.md)
- [GeoIP](actions/GeoIP.md)
- [Grok](actions/Grok.md)
- [Key-Value](actions/KV.md)
- [Remove Tag](actions/RemoveTag.md)
- [Set Field](actions/SetField.md)
- [Unset Field](actions/UnsetField.md)
- [User Agent](actions/UserAgent.md)

*Depending on how log-carver was built, some actions may not be available. Run `log-carver -list-supported` to see the list of actions available in a specific build of log-carver.*

### Conditionals

A conditional starts with an entry with an `if` and a `then`. The pipeline attached to `then` will only be executed for an event if the [Expression](#expression) in the `if` is "truthy".

An `if` conditional can optionally be followed by any number of `else if` with an alternative expression to evaluate and pipeline to execute if the first `if` did not execute. The `else if` does not evaluate its expression if the `if` was executed.

Finally, there can optionally be a single `else`, with the pipeline attached to the `else` itself as opposed to an expression (there is no `then` for an `else`.)

Actions can immediately follow Conditionals and will be executed for all events in that pipeline, and likewise Conditionals can immediately follow Actions.

An example of all three is below.

```yaml
- if: expression
  then:
  # pipeline
- else if: expression
  then:
  # pipeline
- else:
  # pipeline
```

## `receivers`

Array of Receivers. Optional.

The receivers configuration specifies which transports Log Carver should listen on to receive events from.

Events received from a receiver will have a `@metadata[receiver]` entry in them containing several values describing the source of the event:

- `@metadata[receiver][name]`: The [`name`](#name-receiver) of the receiver in configuration
- `@metadata[receiver][listener]`: The listen address of the receiver
- `@metadata[receiver][remote]`: The remote address of the connection
- `@metadata[receiver][desc]`: The description of the connection, usually `-` for TCP connections and the certificate common name or `No Client Certificate` for TLS connections

Each entry has the following properties.

### `enabled` (receiver)

Boolean. Optional. Default: true

Allows a receiver to be disabled without removing it's configuration.

### `listen`

Array of Strings. Required

Sets the list of endpoints to listen on. Accepted formats for each endpoint entry are:

- `ipaddress:port`
- `hostname:port` (A DNS lookup is performed)
- `@hostname` (A SRV DNS lookup is performed, with further DNS lookups if
required)

### `max pending payloads` (receiver)

Number. Optional. Default: 10
Since 2.7.0

Only applicable to protocol-based transports such as "tls" and "tcp" that
support acknowledgements.

The maximum number of spools that can be in process from a connection at any
one time. Each spool will be kept in memory until it is fully processed and
acknowledged by the transport.

If a connection attempts to send more than this many spools to Log Carver,
Log Carver will stop reading from the connection until it has acknowledged
all pending payloads, and will then close the connection, forcing the client
to retry.

*You should only change this value if you changed the equivilant value on a
Log Courier client.*

### `max queue size` (receiver)

Number. Optional. Default: 134217728 (128 MiB)
Since 2.13.0

Maximum number of bytes that can be received and queued from clients at any
one moment in time.

If too many events are being received than can be processed then this queue
can build in size. When this queue is full, when data is received from a
connection that cannot be added to the queue, the data is discarded and the
connection closed.

Warnings will be logged when this happened no more frequently than 1 per
minute to note that events are discarded.

For protocol-based transports that support acknowledgement, no data loss
occurs as the client will know to resubmit the data again on a retried
connection attempt which in the Log Courier case will backoff longer on
each connection attempt to allow Log Carver to catchup.

### `max tls version` (receiver)

String. Optional. Default: ""
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls` or `streamtls`

If specified, limits the TLS version to the given value. When not specified, the TLS version is only limited by the versions supported by Golang at build time. At the time of writing, this was 1.3.

### `min tls version` (receiver)

String. Optional. Default: 1.2
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls` or `streamtls`

Sets the minimum TLS version allowed for connections on this transport. The TLS handshake will fail for any connection that is unable to negotiate a minimum of this version of TLS.

### `name` (receiver)

String. Optional. Default: Empty string

Sets a name for the receiver that will be added to all events under `@metadata[receiver][name]`.

### `ssl certificate` (receiver)

Filepath. Required when `transport` is `tls` or `streamtls`

Path to a PEM encoded certificate file to use as the server certificate.

This is the counterpart of Log Courier's [`ssl ca`](../log-courier/Configuration.md#ssl-ca).

NOTE: SHA1 signed certificates will be no longer supported for security reasons
from version 2.10.0. Setting the environment variable `GODEBUG` to `x509sha1=1`
will temporarily enable support, but users should update their certificates.

### `ssl client ca` (receiver)

Array of Filepaths. Optional
Available when `transport` is `tls` or `streamtls`

A list of paths to PEM encoded client certificate authorities that can be used to verify client certificates. This is the counterpart to Log Courier's [`ssl certificate`](../log-courier/Configuration.md#ssl-certificate).

NOTE: SHA1 signed certificates will be no longer supported for security reasons
from version 2.10.0. Setting the environment variable `GODEBUG` to `x509sha1=1`
will temporarily enable support, but users should update their certificates.

### `ssl key` (receiver)

Filepath. Required when `transport` is `tls` or `streamtls`

Path to a PEM encoded private key to use with the server certificate.

### `transport` (receiver)

String. Optional. Default: "tls"  
Available values: "tls", "tcp", "streamtls", "stream"

*Depending on how log-carver was built, some transports may not be available. Run `log-carver -list-supported` to see the list of transports available in a specific build of log-carver.*

Sets the transport to use when receiving logs from the endpoint.

"tls" listens for TLS encrypted connections using the Courier protocol. For example, it will receive connections from Log Courier and also from the `logstash-output-courier` Logstash plugin. The `ssl certificate` and `ssl key` options are required for this transport. You can enable client certificate authentication by specifying the certificate authority for the client certificates to trust in `ssl client ca`.

"streamtls" listens for TLS encrypted connections that are lined based. For example, it can receive connections from Monolog's [SocketHandler](https://github.com/Seldaek/monolog/blob/main/src/Monolog/Handler/SocketHandler.php) and generate basic events for each line of data received on the connection. The `ssl certificate` and `ssl key` options are required for this transport. If your client supports client certificate authentication (Monolog's SocketHandler does not), this can be enabled by specifying the certificate authority for the client certificates to trust in `ssl client ca`.

"tcp" and "stream" are **insecure** equivalents to "tls" and "streamtls" that do not encrypt traffic or authenticate the identity of endpoints. These should only be used on trusted internal networks. If in doubt, use the secure authenticating transports "tls" and "streamtls". They have no required options.

### `verify peers` (receiver)

Boolean. Optional. Default: true
Available when `transport` is `tls` or `streamtls`

When `ssl client ca` entries are configured for client certificate verification, the default is to require all connections to provide a client certificate and to be verified. If this is set to false, clients will be able to connect without providing a client certificate or with any client certificate.

This setting is ignored if `ssl client ca` is empty.

**Setting this to false is insecure and should only be used for testing.**
