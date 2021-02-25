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
  - [`network`](#network)
    - [`failure backoff`](#failure-backoff)
    - [`failure backoff max`](#failure-backoff-max)
    - [`index pattern`](#index-pattern)
    - [`max pending payloads`](#max-pending-payloads)
    - [`method`](#method)
    - [`reconnect backoff`](#reconnect-backoff)
    - [`reconnect backoff max`](#reconnect-backoff-max)
    - [`retry backoff`](#retry-backoff)
    - [`retry backoff max`](#retry-backoff-max)
    - [`rfc 2782 srv`](#rfc-2782-srv)
    - [`rfc 2782 service`](#rfc-2782-service)
    - [`routines`](#routines)
    - [`servers`](#servers)
    - [`ssl ca`](#ssl-ca)
    - [`ssl certificate`](#ssl-certificate)
    - [`ssl key`](#ssl-key)
    - [`template file`](#template-file)
    - [`template patterns`](#template-patterns)
    - [`timeout`](#timeout)
    - [`transport`](#transport)
  - [`pipelines`](#pipelines)
    - [Actions](#actions)
    - [Conditionals](#conditionals)
  - [`receivers`](#receivers)
    - [`enabled` (receiver)](#enabled-receiver)
    - [`listen`](#listen)
    - [`ssl certificate` (receiver)](#ssl-certificate-receiver)
    - [`ssl key` (receiver)](#ssl-key-receiver)
    - [`ssl client ca` (receiver)](#ssl-client-ca-receiver)
    - [`verify peers` (receiver)](#verify-peers-receiver)
    - [`min tls version` (receiver)](#min-tls-version-receiver)
    - [`max tls version` (receiver)](#max-tls-version-receiver)
    - [`transport` (receiver)](#transport-receiver)

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

```json
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
Available when `transport` is `es`

Pause this long before retrying a bulk operation on Elasticsearch. If the remote endpoint is overwhelmed, this slows down the rate of bulk indexing attempts. On each consecutive failure, the pause is exponentially increased.

When set to 0, the initial retry attempt is made immediately. The second attempt then pauses for 1 second and begins to exponentially increase on each consecutive failure.

### `retry backoff max`

Duration. Optional. Default: 300s  
Available when `transport` is `es`

The maximum time to wait between retry attempts. This prevents the exponential increase of `retry backoff` from becoming too high.

### `rfc 2782 srv`

Boolean. Optional. Default: true

When performing SRV DNS lookups for entries in the [`servers`](#servers) list,
use RFC 2782 style lookups of the form `_service._proto.example.com`.

### `rfc 2782 service`

String. Optional. Default: "courier"

Specifies the service to request when using RFC 2782 style SRV lookups. Using
the default, "courier", an "@example.com" endpoint entry would result in a
lookup for `_courier._tcp.example.com`.

### `routines`

Number. Optional. Default: 4. Min: 1. Max: 32
Available when `transport` is `es`

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

Filepath. Required  
Available when `transport` is one of: `tls`

Path to a PEM encoded certificate file to use to verify the connected endpoint.

### `ssl certificate`

Filepath. Optional  
Available when `transport` is one of: `tls`

Path to a PEM encoded certificate file to use as the client certificate.

### `ssl key`

Filepath. Required with `ssl certificate`  
Available when `transport` is one of: `tls`

Path to a PEM encoded private key to use with the client certificate.

### `template file`

String. Optional
Available when `transport` is `es`

Specify a path to an index template to be installed instead of the builtin one that Log Carver provides. See [Index Templates](#index-templates) for more information.

### `template patterns`

Array of Strings. Optional. Default: Single item of "logstash-*"
Available when `transport` is `es`

When using the builtin index template that Log Carver provides, this can be used to override the indices patterns that the template will apply to. See [Index Templates](#index-templates) for more information.

Ignored if `template file` is specified, as you can specify the patterns inside the custom template.

### `timeout`

Duration. Optional. Default: 15

This is the maximum time Log Carver will wait for a endpoint to respond to a
request after logs were send to it. If the endpoint does not respond within this
time period the connection will be closed and reset.

### `transport`

String. Optional. Default: "tls"  
Available values: "tcp", "tls", "es"

*Depending on how log-carver was built, some transports may not be available. Run `log-carver -list-supported` to see the list of transports available in a specific build of log-carver.*

Sets the transport to use when sending logs to the endpoints. "es" is recommended for most users.

"es" sends events to an Elasticsearch cluster. It will also install a template called `logstash` if one does not exist. If you are migrating from Logstash this template will already exist. It should be compatible if you haven't used the Log Courier `enable ecs` configuration.

"tcp" is an **insecure** equivalent to "tls" that does not encrypt traffic or
authenticate the identity of endpoints. This should only be used on trusted
internal networks. If in doubt, use the secure authenticating transport "tls".

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

A conditional starts with an entry with an `if` and a `then`. The pipeline attached to `then` will only be executed for an event if the [Expression](#expressions) in the `if` is "truthy".

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

### `ssl certificate` (receiver)

### `ssl key` (receiver)

### `ssl client ca` (receiver)

### `verify peers` (receiver)

Boolean. Optional. Default: true
Available when `transport` is `tls`

### `min tls version` (receiver)

String. Optional. Default: 1.2
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls`

Sets the minimum TLS version allowed for connections on this transport. The TLS handshake will fail for any client that is unable to negotiate a minimum of this version of TLS.

### `max tls version` (receiver)

String. Optional. Default: ""
Available values: 1.0, 1.1, 1.2, 1.3
Available when `transport` is `tls`

If specified, limits the TLS version to the given value. When not specified, the TLS version is only limited by the versions supported by Golang at build time. At the time of writing, this was 1.3.

### `transport` (receiver)

String. Optional. Default: "tls"  
Available values: "tcp", "tls"

*Depending on how log-carver was built, some transports may not be available. Run `log-carver -list-supported` to see the list of transports available in a specific build of log-carver.*

Sets the transport to use when receiving logs from the endpoint.

"tcp" is an **insecure** equivalent to "tls" that does not encrypt traffic or
authenticate the identity of endpoints. This should only be used on trusted
internal networks. If in doubt, use the secure authenticating transport "tls".
