# Change Log

## 2.6.0

23rd March 2021

Log Courier

- Fix broken `includes` configuration that was broken in 2.5.0 and add preventative tests
- Added new `reader` configuration to `files` entries that defaults to `"line"`
- Added a `"json"` `reader` that can read JSON files containing objects without line separators or line endings and decode them into events

Log Carver

- Improved speed of root level field lookups in expressions

## 2.5.6

17th February 2021

Log Courier

- Resolve crash with orphaned file processing

## 2.5.5

9th February 2021

Log Courier

- Fix severe registrar corruption that prevented Log Courier from resuming files at the correct offset
- Fixed a crash when `add offset field` is set to false and `enable ecs` is set to true
- Implement `hold time` configuration option with a default of 96 hours. Log Courier will now, by default, only hold open deleted files for a maximum of 96 hours after deletion is detected, regardless of whether its contents finish processing. A warning is logged if data has been lost when the file closed. This ensures disks do not fill when the pipeline is blocked. 96 hours was chosen as the default to allow a minimum of a few days to detect and repair a pipeline issue, as some roll over configurations delete the file during the very first rollover to replace it with a compressed version.
- The `dead time` configuration will no longer be checked if the pipeline is completely blocked. Previously, it would be processed during complete pipeline blockage only, meaning a deleted file could be closed and data lost if the pipeline was completely blocked for the specified `dead time` period. This was unintended behaviour and would not trigger if the pipeline was extremely slow as the `dead time` would reset upon each successful read. Documentation has also been updated to clarify that `dead time` is not based on the modification time of the file, but the time of the last successful read when the pipeline is moving, however slow that may be.

## 2.5.4

8th February 2021

Log Courier

- Resolve `lc-admin` regression with unix sockets - it now reports `unix-socket` instead of `log-courier-address`

## 2.5.3

8th February 2021

Log Courier

- Fix `lc-admin` missing default configuration on RPM builds
- Fix `lc-admin` reporting unknown configuration options it should never be concerned with
- Fix `lc-admin` reporting `log-courier-address` as the address in errors

## 2.5.2

7th February 2021

Log Courier / Log Carver

- Fix integer overflow preventing 32-bit compilation

## 2.5.1

7th February 2021

Log Courier

- Restore backwards compatibility with v2.0.6 by removing ECS fields that were added in v2.5.0
- New configuration option to enable ECS fields: `enable ecs`
- ECS fields now also obey the corresponding `add xxx` configuration options

Log Courier / Log Carver

- Fix vendoring issue of a dependency that was blocking package builds
- No binary changes to Log Courier / Log Carver

## 2.5.0

6th February 2021

Log Courier

- Fix a few issues that could cause Log Courier to hang unnecessarily during shutdown
- Improvements to configuration parsing
- Optimised registrar to only write periodically and not constantly
- Buffers and compression streams are reused during event transmission to further reduce memory and garbage collection
- Rebuilt the transport layer to use less routines per connection and negotiate a new method of event transmission that uses less memory
- Rebuilt application initialisation to allow creation of separate binaries for different tasks running under the same pipelining principle and using similar transports (e.g. Log Carver)
- Add option to enable ECS (Elastic Common Schema) for the builtin fields such as host and file path
- Many more under-the-hood changes to make code more straight forward and to allow code sharing with Log Carver

Log Carver

- Initial beta release
- Can be used as a low memory/CPU substitute to Logstash for basic events
- Supported processor actions include: date, geoip, user_agent, kv, add_tag, remove_tag, set_field, unset_field
- The set_field action supports the Common Expression Language (CEL) for code-like expression support when setting fields
- If/ElseIf/Else support in the pipeline using Common Expression Language (CEL) for code-like conditional expressions
- Receives events over the Log Courier protocol from Log Courier clients
- New ES transport to allow events to be sent directly to Elasticsearch
- Templates embedded for ES6+ that will automatically be inserted as "logstash"
- It is recommended to use new indices and remove the "logstash" template from ES as fields are different and now follow ECS (Elastic Common Schema)
- Configuration documentation is minimal but a minimal example can be found in the docs/examples folder, more will be added in time

## 2.0.6

*9th May 2020*

* Fix several issues when shutting down with outstanding payloads, to prevent
hanging forever

## 2.0.5

18th February 2017

- Fix -stdin run mode not being able to exit cleanly (#353)
- Fix address pool rotation so that the same connection address is used on every
connection attempt
- Fix certificate problem notices showing invalid year values
- Fix end-to-end tests

## 2.0.4

11th June 2016

Log Courier

- Fix random recovery failure when Logstash is unavailable (#324)
- Fix systemd unit files User= declaration that does not support variables. In
order to change the user that Log Courier runs as, edit the unit file directly
after updating to 2.0.4 (#322)
- Fix max pending payloads exceeded when configuration is reloaded or when there
are multiple Logstash connections and one of the connections recovers
- Fix service start failure after a reboot on systems where /var/run is tmpfs
(#321)
- Fix shipping encoding error when configuration file is YAML and global fields
or per-file declaration fields contain nested maps (#325)
- Add the ability to set the group that log-courier runs as

RPM Packaging

- Removed the dependency on ZeroMQ 3 as this was deprecated in 2.x of Log
Courier

## 2.0.3

9th May 2016

- Fix issue where pending payload limit is exceeded when Logstash is overloaded
(#315)
- Fix incorrect stale bytes count in API / `lc-admin`
- Fix the harvester status not updating correctly when reaching end of file in
API / `lc-admin`
- Reintroduce full `status` command into `lc-admin`

## 2.0.2

8th May 2016

- Fix a rare hang when endpoint failure occurred (#314)
- Add a new `debug` command to the REST API and `lc-admin` tool which gives a
live stack trace of the application (#315)

## 2.0.1

25th April 2016

- Fix `lc-admin` ignoring -config parameter and not auto-loading the default
configuration in RPM and DEB packages (#303)
- Fix `lc-admin` numerical outputs such as file completion percentage showing as
1.00e2 instead of 100 (#304)
- Fix hang that could occur when Logstash failed in loadbalance or failover
network methods (#311)
- Fix `admin` `listen address` not having a default (#307)
- Improve documentation for `lc-admin` (#307)
- Improve debug logging of backoff calculations

*NOTE: Please note the building from source step for `go generate` was changed
in 2.0.1.*

## 1.8.3

21st April 2016

Log Courier

- Do not open dead files on startup (which causes excessive memory usage if
there are many of them) if their size has not changed since they were last
opened (#242)
- When a long line was split, the "splitline" tag was not being appended if the
fields configuration already contained tags

RPM Packaging

- Fix broken systemd configuration (#237)
- Fix non-fatal %postun script failure, and fix starting of Log Courier during
package upgrade even if it was not originally started (#228)

DEB Packaging

- Fix broken systemd configuration (#237)

## 2.0.0

20th April 2016

Log Courier 2.x is compatible with the 1.x Logstash plugins.

Breaking Changes

- CurveZMQ transport has been removed
- The `lc-curvekey` utility has been removed
- Multiline codecs can now be configured with multiple patterns. As such, the
`pattern` configuration has been replaced with `patterns` and it is now an array
of patterns
- Multiline codecs have a new `match` configuration that can be set to `all` or
`any`, to control how multiple patterns are used. The default is `any`.
- Multiline patterns can be individually negated with a "!" prefix. The `negate`
configuration directive has been removed. A "=" prefix is also possible to allow
patterns that need to start with a literal "!"
- Filter codec patterns now also accept negation in the same method. The
`negate` configuration directive has also been removed for filters and a new
`match` configuration added that also defaults to `any`
- Multiple codecs can now be specified. As such, the `codec` configuration has
been renamed to `codecs` and must now be an array.
- The `persist directory` configuration is now required, unless it was built in
at build time (which it will be for RPM and DEB packages - see Build Changes).
- All `admin` prefixed configurations have been moved from the `general` section
into their own `admin` section, and the prefix removed
- The `reconnect` transport configuration has been removed, and replaced with
`reconnect backoff` and `reconnect backoff max`

Changes

- A new `global fields` configuration is available in the `general` section
where fields that are to be added to all events from all paths can be specified.
This complements the current `fields` configuration that is per-path.
- A new `method` configuration directive has been added to the `network`
section, which allows `random`, `failover` or `loadbalance` network modes.
Information on these new values can be found in the configuration documentation.
The default is `random` for backwards compatibility.
- The `dead time` stream configuration directive now defaults to 1 hour
- New backoff configurations, `failure backoff` and `failure backoff max`, have
been added, to allow backoff from problematic remote endpoints where, for
example, connection always succeeds but transmission attempts always timeout.
- Configuration files can now be in YAML format by giving the file a `.yaml`
extension, and this will be the preferred format. JSON format will continue to
be used for `.conf` and `.json` configuration files
- The `config` parameter can now be omitted from `log-courier` if a default
configuration file was specified during build (see Build Changes)
- `lc-admin` can now be given a configuration file to load the
`admin`.`listen address` from using the `config` parameter. In the absense of
both `connect` and `config` parameters it will load the default configuration file if
one was specified during build (see Build Changes)
- Log Courier remote administration via `lc-admin` is now a REST interface,
allowing third-party integrations and monitoring
- Harvester snapshot information reported by `lc-admin` now contains additional
information such as the last known file size and percentage completion (#239)
- Harvester snapshot information is now updated even when the remote server is
down
- Publisher snapshot information reported by `lc-admin` now contains a list of
enabled endpoints and their statuses (#199)
- If the harvester detects a log file has had data written without a new line
ending, it will report a warning to the log so it can be investigated

Fixes

- Do not open dead files on startup (which causes excessive memory usage if
there are many of them) if their size has not changed since they were last
opened (#242)
- When a long line was split, the "splitline" tag was not being appended if the
fields configuration already contained tags

RPM Packaging

- Fix broken systemd configuration (#237)
- Fix non-fatal %postun script failure, and fix starting of Log Courier during
package upgrade even if it was not originally started (#228)

DEB Packaging

- Fix broken systemd configuration (#237)

Build Changes

- Some configuration parameters, such as default configuration file and default
persist directory can now be specified at build time by setting environment
variables for them. For example: `LC_DEFAULT_CONFIGURATION_FILE` and
`LC_DEFAULT_GENERAL_PERSIST_DIR`. These will be used in the RPM and DEB packages
to set platform specific defaults
- Simplified the entire build process to use only Go tooling to make
cross-platform building significantly easier, especially on Windows
- Discarded Ruby test framework in favour of developing a Go test framework over
time

## 1.8.2

30th October 2015

Log Courier

No Changes

RPM Packaging

- Fixed incorrect license specification - the spec file specified GPL which was
incorrect - the license is Apache. The spec file has been updated.

Logstash Plugins

- Introduce compatibility with Logstash 2.0.0 (#245)

## 1.8.1

7th August 2015

Log Courier

No Changes

DEB Packaging

- Fix a regression in the upstart script due to the addition of a configtest
(#217 - Thanks to @eliecharra)

Logstash Plugins

- Fix a regression with `ssl_verify_ca` (#214)
- Fix a regression with the output plugin `hosts` setting (#215)

## 1.8

6th August 2015

- Fix various causes of multiline codec causing Log Courier to crash (#188)
- Improve handling of file truncation when using codecs (derived from #194)
- Fix "Unknown message received" errors caused by partial reads which occur
frequently with Logstash 1.5.3 due to OpenSSL cipher hardening (#208)
- Fix systemd service configuration providing incorrect arguments to log-courier
(#204)
- Implement configuration test before start and restart for Debian upstart init
script (#189)
- Implement options to enable/disable automatic fields such as "host", "offset"
and "path"
- Implement an option to add a "timezone" field to events containing the local
machine's timezone in the format "-0700 MST" (#203)

## 1.7

3rd June 2015

- Report a configuration error if no servers are specified instead of continuing
and crashing during startup (#149)
- Report a configuration error if the configuration file is empty or contains
only whitespace instead of crashing during startup (#148)
- Report a configuration error when a files entry is specified without any paths
instead of continuing and ignoring it (#153)
- Implement server shuffling when multiple servers are specified to prevent
site-wide convergence of Log Courier instances towards a single server (#161)
- Fix pending payload limit being exceeded if a disconnect and reconnect occurs
- Fix a rare race condition in the tcp and tls transport that could cause Log
Courier to stop sending events after a connection failure
- Improve build process to work seamlessly with msysGit on Windows

## 1.5

28th February 2015

Breaking Changes

- The way in which logs are read from stdin has been significantly changed. The
"-" path in the configuration is no longer special and no longer reads from
stdin. Instead, you must now start log-courier with the `-stdin` command line
argument, and configure the codec and additional fields in the new `stdin`
configuration file section. Log Courier will now also exit cleanly once all data
from stdin has been read and acknowledged by the server (previously it would
hang forever.)
- The output plugin will fail startup if more than a single `host` address is
provided. Previous versions would simply ignore additional hosts and cause
potential confusion.

Changes

- Implement random selection of the initial server connection. This partly
reverts a change made in version 1.2. Subsequent connections due to connection
failures will still round robin.
- Allow use of certificate files containing intermediates within the Log Courier
configuration. (Thanks @mhughes - #88)
- A configuration reload will now reopen log files. (#91)
- Implement support for SRV record server entries (#85)
- Fix Log Courier output plugin (#96 #98)
- Fix Logstash input plugin with zmq transport failing when discarding a message
due to peer_recv_queue being exceeded (#92)
- Fix a TCP transport race condition that could deadlock publisher on a send()
error (#100)
- Fix "address already in use" startup error when admin is enabled on a unix
socket and the unix socket file already exists during startup (#101)
- Report the location in the configuration file of any syntax errors (#102)
- Fix an extremely rare race condition where a dead file may not be resumed if
it is updated at the exact moment it is marked as dead
- Remove use_bigdecimal JrJackson JSON decode option as Logstash does not
support it. Also, using this option enables it globally within Logstash due to
option leakage within the JrJackson gem (#103)
- Fix filter codec not saving offset correctly when dead time reached or stdin
EOF reached (reported in #108)
- Fix Logstash input plugin crash if the fields configuration for Log Courier
specifies a "tags" field that is not an Array, and the input configuration for
Logstash also specified tags (#118)
- Fix a registrar conflict bug that can occur if a followed log file becomes
inaccessible to Log Courier (#122)
- Fix inaccessible log files causing errors to be reported to the Log Courier
log target every 10 seconds. Only a single error should be reported (#119)
- Fix unknown plugin error in Logstash input plugin if a connection fails to
accept (#118)
- Fix Logstash input plugin crash with plainzmq and zmq transports when the
listen address is already in use (Thanks to @mheese - #112)
- Add support for SRV records in the servers configuration (#85)

Security

- SSLv2 and SSLv3 are now explicitly disabled in Log Courier and the logstash
courier plugins to further enhance security when using the TLS transport.

## 1.3

2nd January 2015

Changes

- Added support for Go 1.4
- Added new "host" option to override the "host" field in generated events
(elasticsearch/logstash-forwarder#260)
- The Logstash input gem can now be requested to add extra fields to events for
peer identification. The tls and tcp transports can now add a "peer" field
containing the host and port, and the tls transport a "peer_ssl_cn" field that
will be set to the client certificates common name. The "add_peer_fields" plugin
option will enable these fields (#77)
- Fix missing file in Logstash gem that prevents ZMQ transports from working
(#75)
- Fix Logstash gem crash with ZMQ if a client enters idle state (#73)
- During shutdown, Logstash gem with ZMQ will now never loop with context
terminated errors (#73)
- Significantly improve memory usage when monitoring many files (#78)
- Fix Logstash courier output plugin hanging whilst continuously resending
events and add regression test
- Fix Logstash courier output plugin not verifying the remote certificate
correctly
- Various other minor tweaks and fixes

Known Issues

- The Logstash courier output plugin triggers a NameError. This issue is fixed
in the following version. No workaround is available.

## 1.2

1st December 2014

Changes

- Fix repeated partial Acks triggering an incorrect flush of events to registrar
- Fix a loop that could occur when using ZMQ transport (#68)
- TLS and TCP transport will now round robin the available server addresses
instead of randomising
- Implemented "Dead time in" on `lc-admin` harvester statuses
- `lc-admin` status output is now sorted and no longer in random order
- Add a workaround for logstash shutdown looping with "Context is terminated"
messages (#73)
- Implement asynchronous ZMQ receive pipeline in the Logstash gem to resolve
timeout issues with multiple clients and a busy pipeline
- Implement multithreaded SSL accept in the Logstash gem to prevent a single
hung handshake attempt from blocking new connections
- Switch to ruby-cabin logging in the gems to match Logstash logging
- Updated the RedHat/CentOS 5/6 SysV init script in contrib to follow Fedora
packaging guidelines
- Provided a RedHat/CentOS 7 systemd service configuration in contrib (with
fixes from @matejzero)

Known Issues

- The Logstash courier output plugin hangs whilst continuously resending events.
This issue is fixed in the following version. No workaround is available.

## 1.1

30th October 2014

- Implement gems for the new Logstash plugin system (#60)
- Fix gem build failing on develop branch with old rubygems versions due to a
malformed version string (#62)
- Fix ZeroMQ transports in the ruby gem with Logstash 1.4.x (#63)
- Fix build issue with ZeroMQ 3.2 and `make with=zmq3`
- Fix partial acknowledgements not being passed to registrar and persisted to
disk
- Fix a race condition when the spooler flushes to prevent a timeout occurring
one or more times after a flush due to size
- Print informational messages containing ZMQ library version information during
gem and log-courier startup to aid in diagnostics
- Raise a friendly error when trying to use the zmq transport in the Log Courier
gem with incompatible versions of libzmq
- Various fixes and improvements to log-courier, gem, build and tests

## 1.0

23rd October 2014

- Remove `ping` command from `lc-admin` (#49)
- Empty lines in a log file are incorrectly merged with the following line (#51)
- Don't require a connection to Log Courier when running `lc-admin help` (#50)
- Bring back `make selfsigned` to quickly generate self-signed TLS certificates
(#25)
- Implement `make curvekey` to quickly generate curve key pairs (#25)
- Fix hanging ZMQ transport on transport error
- Fix timeout on log-courier side when Logstash busy due to non-thread safe
timeout timer in the log-courier gem
- Gracefully handle multiline events greater than 10 MiB in size by splitting
events

## 0.15

23rd September 2014

- Fix admin being enabled by default when it shouldn't be (#46)

## 0.14

18th September 2014

Breaking Changes

- The 'file' field in generated events has been renamed to 'path'. This
normalises the events with those generated by Logstash itself, and means the
Logstash `multiline` filter's default `stream_identity` setting is compatible.

Changes

- Fix connection failure and retry sometimes entering an unrecoverable state
that freezes log shipping
- Fix ProtocolError with large log packets and on idle connections (since 0.13)
- Provide more information when the gem encounters ProtocolError failures
- Fix ssl_verify usage triggering error, "Either 'ssl_verify_default_ca' or
'ssl_verify_ca' must be specified when ssl_verify is true" (#41)
- Fix previous_timeout multiline codec setting (#45)
- Restore message reliability and correctly perform partial ack. Since 0.9
events from log-courier could be lost after a broken connection and not
retransmitted
- Significantly improve Log Courier gem performance within JRuby by switching
JrJackson parse mode from string to raw+bigdecimal
- Add unix domain socket support to the administration connection
- Provide publisher connection status via the administration connection
- Gracefully handle lines greater than 1 MiB in size by splitting and tagging
them, and make the size configurable (#40)

Known Issues

- Admin is enabled by default when it shouldn't be. Workaround: Set the
"admin enabled" general configuration option to false.

## 0.13

30th August 2014

- Added new administration utility that can connect to a running Log Courier
instance and report on the current shipping status
- Added new filter codec to allow selective shipping and reduce Logstash loads
- Fixed Logstash plugin entering infinite loop during Logstash shutdown sequence
when using ZMQ. The plugin now shuts down gracefully along with Logstash (#30)
- Fixed unexpected registrar conflict messages appearing for a short time after
a log rotation occurred (#34)
- Fixed Logstash crashing with "Operation cannot be accomplished in current
state" when using ZMQ and Logstash hits a bottleneck requiring partial ACKs to
be sent to Log Courier
- Improved performance of the Log Courier Logstash plugins
- Various other minor rework and improvements

## 0.12

4th August 2014

- Fix non-ASCII but valid UTF-8 characters getting replaced with question marks
by the Logstash gem (#22)
- Fix zmq transport not working in Logstash due to ffi-rzmq version too old. Gem
installation will now trigger update of the ffi-rzmq gems to the necessary
versions (#20)
- Fix broken syslog logging (#18)
- Fix broken spool-size configuration setting (#17)
- Fix compilation on Windows (#23)
- Fix shutdown not working when publisher has pending payloads (#24)
- Fix potential race condition issues in the ZMQ Logstash plugin
- Implement ZMQ monitor and log when connections/disconnects happen
- Move logging cmdline settings (such as log-to-syslog) to the configuration
file and allow configuration of stdout logging and file logging as well as
syslog logging (#15)
- Remove support for Go 1.1 due to json.Marshal returning error InvalidUTF8Error
on encountering an invalid sequence. Go 1.2 and above do not and replace invalid
sequence with the Unicode replacement character
- Various other minor tweaks and fixes

## 0.11

13th July 2014

- Security fix: Ruby gem client (used by Logstash output plugin) did not verify
 certificate hostname
- Fix edge case file rotation problems (#7)
- Fix incorrect field population in events (#9)
- Fix random hang when Log Courier loses connection to Logstash (#8)
- Improve logging and make the level of detail configurable (#10)
- Fix PING/PONG protocol errors when idle (#12)
- Move spool-size and idle-timeout to the configuration file as "spool size" and
"spool timeout"
- Replace `make selfsigned` with a utility lc-tlscert that can generate
self-signed certificates and the necessary log snippets like genkey did for ZMQ
- Rename genkey to lc-curvekey for consistency
- Various other minor tweaks and fixes

## 0.10

29th June 2014

- Support for Go 1.3 (#3)
- Configuration can be reloaded while log courier is running by sending the
SIGHUP signal (\*nix only)
- Additional configuration files can be imported by the main configuration file
using the new `"includes"` section which is an array of Fileglobs. (#5)
- Added `make selfsigned` to allow quick generation of SSL certificates for
testing
- A `"general"` section has been added to the configuration file
- The directory to store the .log-courier persistence file can now be
configured under `"general"/"persist directory"`.
- How often the filesystem is examined for log file appearances or movements
can now be configured under `"general"/"prospect interval"`.
- Fix gem build instructions (#6)
- Fix instances where a file entry has multiple `"fields"` entries results in
all fields having the same value as the first field. (#4)

## 0.9

14th June 2014

- Restructure and tidy the project and implement new build tools
- Rename to **Log Courier**
- Implement a completely new test framework with even more tests
- Introduce a new protocol and TLS transport layer that is faster on high
latency links such as the internet
- Implement support for a CurveZMQ transport to allow transmission of logs to
multiple servers simultaneously at low latency.
- Improve efficiency of the event spooler
- Greatly improve the Logstash plugin
- If a log file cannot be opened, only retry as long as the log file exists and
not forever
- Fix offset field on events to point to the start of the event, not the end
- Enable comments inside the configuration file
- Reduce unnecessary logging

## Pre-0.10

The following are fixes present in the Driskell fork of Logstash Forwarder 0.3.1
which Log Courier builds upon.

- Fix state persistence when following multiple files
- Improve log rotation handling
- Make reconnect frequency configurable
- Start from beginning of a log file if created after startup
- Implement partial ACK support into the transmission protocol to prevent
timeout issues and "Too many open files" crashes on remote Logstash instances
- Fix rotation detection on Windows
- Prevent log files from getting locked during harvesting on Windows that
prevents the logging program from renaming the file
- Add a codec system to allow events to be pre-processed before transmission. A
multiline codec to combine multiple lines into single events is now available.
- Fix test suite and add new tests
- Add support for FreeBSD
(<https://github.com/elasticsearch/logstash-forwarder/pull/132> by https://github.com/atwardowski)
- Fix duplicated log file import caused by incomplete lines in the log file
(<https://github.com/elasticsearch/logstash-forwarder/pull/164> by https://github.com/tzahari)
- Add support for newer SSL certificate types
(<https://github.com/elasticsearch/logstash-forwarder/pull/188> by https://github.com/pilif)
- Add support for IPv6 servers
(<https://github.com/elasticsearch/logstash-forwarder/pull/143> by https://github.com/yath)
