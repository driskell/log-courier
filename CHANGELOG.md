# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

## [2.12.0](https://github.com/driskell/log-courier/compare/v2.11.0...v2.12.0) (2025-02-18)


### Features

* Allow immediate retry of failed files on reload ([25ec0cf](https://github.com/driskell/log-courier/commit/25ec0cfe2d476d68778a4d97a72f8b62a745ff8b))


### Bug Fixes

* Prevent permanent failures in harvester such as max line bytes exceeded from causing errors every prospector scan loop ([3f3ea20](https://github.com/driskell/log-courier/commit/3f3ea209ea9456247cc5c37c1e9584db4ecdcec3))

## [2.11.0](https://github.com/driskell/log-courier/compare/v2.10.0...v2.11.0) (2024-07-30)


### Features

* Added "json" action to decode field and copy resulting fields into root of event ([fba6388](https://github.com/driskell/log-courier/commit/fba6388e82e5be69ae524682a9df3e6c3d5cd4f1))
* Combine similiar event index failures, reporting only first message and total count ([4a148b3](https://github.com/driskell/log-courier/commit/4a148b353835a4b51ad04b888f7a353db2a51a0a))
* Grok action will now only remove the field if grok parsing was successful ([676225e](https://github.com/driskell/log-courier/commit/676225e02a5aac58dfe24b1a1683e87d51b529b4))
* Implement [@metadata](https://github.com/metadata)[receiver][name] and others against all events to allow processors to process based on connection details (this replaced agent[source] which is removed) ([9ba6ba7](https://github.com/driskell/log-courier/commit/9ba6ba79429393d7b2b26d4da86d4f3f916e4761))
* Log Carver now appends direct source of a received event to agent[source] ([68b88c7](https://github.com/driskell/log-courier/commit/68b88c774ec02431df0ea34389737a6812a6f74b))
* The [@timestamp](https://github.com/timestamp) field can now be set from grok and json actions if in the correct format expected (RFC3339 with optional nanoseconds: 2006-01-02T15:04:05.999Z) ([36bc467](https://github.com/driskell/log-courier/commit/36bc4674f5181f8f0e550a77756badfa8e461b73))
* TLS handshake errors are now prefixed to show they originate from a handshake ([1d8a57f](https://github.com/driskell/log-courier/commit/1d8a57f3f15224b251a5162ffa123c52cfb8baf3))


### Bug Fixes

* Fix a rare race in publisher for immediately failing reconnections that can cause high CPU and pipeline freeze ([e3aacc8](https://github.com/driskell/log-courier/commit/e3aacc8c2c3cd99f351331df0660bc7669a3bcfd))
* Fix crash if reconnection reusing port occurs ([d5c1cc2](https://github.com/driskell/log-courier/commit/d5c1cc29c20392688b6d50a4022a0ed82501635e))
* Fix potential log-carver deadlock when using lc-admin -carver and a new connection appears during the status poll ([d702ce1](https://github.com/driskell/log-courier/commit/d702ce1710fd7677e21c0cde9012ab8319b1e31d))
* Fix potential panic in receiver if a connection is lost ([3615aa3](https://github.com/driskell/log-courier/commit/3615aa39693af2b705e9c988cfb04d16df7ce643))
* Fix receiver pool not cleaning up connections in some instances and preventing shutdown, and add additional logs ([f072865](https://github.com/driskell/log-courier/commit/f072865b1e0a6fd7ac5d643b28ceb73f8732f638))
* Fix receiver shutting down and reset messages displaying incorrectly ([4d7c873](https://github.com/driskell/log-courier/commit/4d7c873413f4ca91159975275e17d8ef24a03f5f))
* Fix scheduling issues that can cause unexpected timeouts on new connections ([b0980cc](https://github.com/driskell/log-courier/commit/b0980ccd3515f2c096add5b7bf89457118000d6c))
* Fix send congestion or send errors not causing connection shutdown and leaking connections ([02c8ede](https://github.com/driskell/log-courier/commit/02c8ede634e847a4ec1ba2c501ebb674ae56c8a2))
* Fix successful ES bulk request failing with cluster block message due to cluster block handling fix ([20e5751](https://github.com/driskell/log-courier/commit/20e57516d8ed67606cac653a8d1f580c016ed89f))
* Fix timeout on courier displaying two error messages, one for unexpected end ([e453486](https://github.com/driskell/log-courier/commit/e453486ec0edd551eeb0617eb6729ca6e102e5cd))
* Float grok values and conversion to float expressions now serialize correctly to float in the Elasticsearch output ([bb0f4a9](https://github.com/driskell/log-courier/commit/bb0f4a9a2251f5cb24ed38cc689faf4288e25940))
* Receiver can cause panic during reload in Log Carver ([c43f935](https://github.com/driskell/log-courier/commit/c43f93514dd913174cdd0d6ba686af1e3821e5ea)), closes [#400](https://github.com/driskell/log-courier/issues/400)
* Receiver pool can sometimes crash when a disconnect occurs unexpectedly ([3553db5](https://github.com/driskell/log-courier/commit/3553db5e5e684cc4595d5efb2aba916e70ec6c60))
* Resolve a formatting issue in Handshake failure output ([2f6d77d](https://github.com/driskell/log-courier/commit/2f6d77d558546f47e66c08712a28b3677543550a))
* Resolve connection leakage that can cause too many open files, especially with stream input ([39f87a9](https://github.com/driskell/log-courier/commit/39f87a9c500ef2297ff3b72b4aa94cfdf6ca4d60))
* Resolve crash introduced with cleanup code for force failed connections ([a17595b](https://github.com/driskell/log-courier/commit/a17595b1733ef0f06a0e8da6110b13cc31dfefb1))
* Resolve ES template not indexing terms for messages longer than 256 characters ([99bf67f](https://github.com/driskell/log-courier/commit/99bf67f6d69309fa3bba30eb0caf6d98906776ac))
* Resolve force closed connections on receiver not being cleaned up fully, preventing shutdown ([b6689de](https://github.com/driskell/log-courier/commit/b6689de004e840d933cd259f9bc2d9516b30eaeb))
* Resolve linereader hanging on streams due to recent fix on missing lines ([bdc463c](https://github.com/driskell/log-courier/commit/bdc463c95082532f405e6319190dc8a245814748))
* Resolve missing lines at end of streams due to harvester early failure ([710fc3c](https://github.com/driskell/log-courier/commit/710fc3c915f1338351f8b049f9cd18a011e6d1e8))
* Resolve potential crash in receiver cleanup if shutdown of a connection occurs whilst draining ([fcd54d6](https://github.com/driskell/log-courier/commit/fcd54d62d4ac84978e44f48fa18514e8ce755294))
* Resolve potential crash when no endpoints are available to publish to ([463ed63](https://github.com/driskell/log-courier/commit/463ed6354fb1efa95e1d3f1df68ee2879627d589))
* Workaround some EOF errors on connections that closed gracefully ([7d20247](https://github.com/driskell/log-courier/commit/7d202474a0051e0916bf2fbef4b09b39f9d647e5))

## [2.10.0](https://github.com/driskell/log-courier/compare/v2.9.1...v2.10.0) (2023-03-20)


### Features

* As of 2.10.0 SHA1 signed certificates will be no longer supported, but can be temporarily enabled by setting the GODEBUG environment variable to x509sha1=1 ([d7659a3](https://github.com/driskell/log-courier/commit/d7659a35899abd303f4f00c9d949b43f2ba4a874))
* Implement support for /**/ matching in file paths, and report IO errors on first scan ([95daa0d](https://github.com/driskell/log-courier/commit/95daa0d137a87cee060b25b0440d0d0190a7a28d)), closes [#327](https://github.com/driskell/log-courier/issues/327) [#285](https://github.com/driskell/log-courier/issues/285)
* Implement TCP streaming receiver ([b3a3720](https://github.com/driskell/log-courier/commit/b3a37204249c48d96a2dc09750b930a0c0804b86))
* Improved failover of connections when using SRV records ([05fcd48](https://github.com/driskell/log-courier/commit/05fcd4892e20121b9ba2f9a43c8f624fa565d8df))
* Improved log output for transports to display more meaningful connection details in some instances ([1a14888](https://github.com/driskell/log-courier/commit/1a14888a9482a2704a2540d45d8b634d1b96c718))
* lc-admin file list is now sorted to prevent display jumps on highly active instances ([17e9014](https://github.com/driskell/log-courier/commit/17e9014052f8637ef8ca6889264581ef15649907)), closes [#396](https://github.com/driskell/log-courier/issues/396)
* Sort receivers and transports within lc-admin ([74775cc](https://github.com/driskell/log-courier/commit/74775cc189a80ef58b047adb5dcafe2a07f1d268))
* SRV record servers now expand after lookup as if the looked up hosts were listed servers, enabling failover and load balancing support ([3a2fecf](https://github.com/driskell/log-courier/commit/3a2fecfc6803f08f8c75e3a16e9545c28037b749)), closes [#354](https://github.com/driskell/log-courier/issues/354)


### Bug Fixes

* Fix :127.0.0.1:1234 not working as specified in documentation for admin connect string ([ef11492](https://github.com/driskell/log-courier/commit/ef11492d42f6e2ba482dea1b5f59cad733406e9c)), closes [#395](https://github.com/driskell/log-courier/issues/395)
* Fix es-https transport reporting ssl ca required when it was ([1741ba7](https://github.com/driskell/log-courier/commit/1741ba730369d9647bea9eaa6d49685d3318712c))
* Fix for cache of ES transport clients that can result in too many open files ([bf45f4e](https://github.com/driskell/log-courier/commit/bf45f4e0379c3ab7940d208ab12a8a926744b5d2))
* Fix loadbalance not balancing effectively when under pressure and queueing more payloads than it should on single endpoints ([54aef03](https://github.com/driskell/log-courier/commit/54aef03ef4d18e5fa04f1fd82f94e13631eaeebc))
* Fix logstash-input-courier not shutting down with Logstash pipeline ([ea7a63c](https://github.com/driskell/log-courier/commit/ea7a63cc94a11152e9bf1d0f20656ed23bfad88c)), closes [#397](https://github.com/driskell/log-courier/issues/397)
* Fix rare crash due to concurrent encoding activity in transport by preventing encoding making writes to the event ([ab6e5f7](https://github.com/driskell/log-courier/commit/ab6e5f7b9ac5ff1afd3d6d8e712a62fbce5f1b36))
* Fix tls not working correctly for es-https, imporve tls config management, fix verify peers, make cert/key required for carver receiver ([c840355](https://github.com/driskell/log-courier/commit/c840355e8f314822dbf630d9b912d1abc9415c4a))

## [2.9.1](https://github.com/driskell/log-courier/compare/v2.9.0...v2.9.1) (2022-10-15)


### Bug Fixes

* Fix installation issue of gems due to missing version.rb file ([a2dc5e4](https://github.com/driskell/log-courier/commit/a2dc5e4f680f0b66d0abf1e13006ddc240629110)), closes [#393](https://github.com/driskell/log-courier/issues/393)
* Fix recursive lock in harvester blocking admin socket ([dc03b40](https://github.com/driskell/log-courier/commit/dc03b40da03a0b1982418e5f85d33125bc3d3ea6)), closes [#394](https://github.com/driskell/log-courier/issues/394)

## 2.9.0

26th March 2022

Log Carver

- Updated CEL Go library to 0.9.0
- Implemented CEL Go string extensions and base64 encoder
- Implemented JSON encoder (json_encode and json_decode CEL Go functions)
- Fixed lc-admin status for transports not updating in some cases

## 2.8.1

23rd November 2021

Log Courier / Log Carver

- Fixed crash during configuration reload for `tcp` receiver and transport
- Fixed reload of configuration not correctly updating endpoints
- Fixed reload of configuration sometimes causing a deadlock with hugh CPU

## 2.8.0

22nd November 2021

Log Courier / Log Carver

- Fixed transports sometimes not restarting when configuration is changed
- Fix regression caused by fix for #367 that can cause Log Courier to crash on startup

Log Carver

- Added `es-https` transport to use encrypted HTTPS communication with Elasticsearch
- Added `min tls version` and `max tls version` to `es-https` transport
- Added `ssl ca` for `es-https` transport to allow certificate pinning
- Added `username` and `password` for `es` and `es-https` transports to allow Basic authentication scheme
- Improved Elasticsearch bulk response validation
- Fixed missing `RuntimeDirectory` in Log Carver systemd configuration which is necessary for default admin socket listen location

`lc-admin`

- Added `-carver` option to connect to automatically detect the administration socket from Log Carver's configuration file or from known defaults

## 2.7.4

26th October 2021

Logstash Input Plugin

- Fix regression caused by removal of `peer_recv_queue` - some code remained that was still attempting to access it

## 2.7.3

26th October 2021

Log Courier

- Add new [`add timezone name field`](docs/log-courier/Configuration.md#add-timezone-name-field) configuration that adds the timezone name such as `UTC` or `Europe/London` for use with the Logstash Date Filter. The existing `timezone` / `event.timezone` (ECS) fields were in a format the filter could not use (#345)
- Fix race that might cause a file to use a configuration other than the first configuration it matches (#367)

Logstash Input Plugin

- Removed `peer_recv_queue` configuration as it is unused. Only a single payload is received and processed at any one time by the plugin.

Logstash Output Plugin

- Now maintained again and updated to use latest log-courier ruby implementation which includes protocol handshake support
- Added support for `tcp` only output

## 2.7.2

21st October 2021

Logstash Input Plugin

- Fix compatibility issue between Ruby Log Courier and Go Log Courier caused by a discrepency in the HELO/VERS protocol message that form the handshake

## 2.7.1

21st October 2021

Logstash Input Plugin

- Fix missing protocol.rb file causing import error

## 2.7.0

21st October 2021

Log Courier / Log Carver

- Added `last_error` and `last_error_time` to `lc-admin` for endpoints, so that the last error can be inspected
- Added the negotiated TLS version to connection messages and added additional logging where a remote does not support protocol handshakes
- Improved to `random` transport method so that a failed endpoint remains active and retrying until the switch happens, allowing it's status and last error to be seen in `lc-admin` instead of `endpoints: none`
- Improved moving average speed calculations
- Fixed panic in `test` transport when a payload containing a single event is encountered
- Fixed startup hang if the pipeline fails to start, for example when a port is already in use

Log Carver

- Added [`max pending payloads`](docs/log-carver/Configuration.md#max-pending-payloads-receiver) configuration to `receiver` section, to ensure clients cannot DoS Log Carver
- Fix a connection failing during attempt to gracefully shut it down
- Fix a possible deadlock in receiver shutdown due to late acknowledgements for a failed connection during shutdown

`lc-admin`

- **Breaking Change:** Removed the prompt when `lc-admin` is run without arguments and replaced it with an interactive console
- Added screens for monitoring the prospector, receiver and publisher, which refresh every second
- Note that scrolling is not yet implemented and so a larger terminal screen may be required to see all data for busy instances

Logstash Input Plugin

- **Breaking Change:** Obsoleted and removed the `zmq` transport option
- Updated dependencies to newer versions
- Added [`min_tls_version`](docs/LogstashIntegration.md) configuration option that now defaults to 1.2 (#357)
- Added protocol handshake support to output version of connecting clients
- Added new log messages to output the negotiated TLS version of each connection and, where a handshake occurs, the remote's product and version

## 2.6.4

14th October 2021

Log Courier / Log Carver

- Simplified networking logic and fixed some deadlocks in publisher and scheduler
- Improved logging of transports, receivers, endpoints and publisher
- Faster TCP/TLS shutdown if the transport is an unusable state
- The name and version of the remote is now logged for new connections as part of the HELO/VERS handshake

Log Carver

- Implemented controlled shutdown of log-carver's log-courier connections to ensure all received events are acknowledged, so that log-courier does not resend any events already sent to the transport when it reconnects
- Added additional timeouts to ensure that all dead connections to log-carver are detected and closed
- Fixed shutdown hanging forever if an ES transport is unable to retrieve node information

## 2.6.3

20th September 2021

Log Courier

- Fix debug level logging outputting spurious messages (#385)
- Fix syslog entries progname to only have the binary name and not the full path (#384)
- Fix hold time settings not closing files properly and causing a notice every 10 seconds (#382)
- Fix a deadlock in spooler if the pipeline completely stopped

Log Carver

- Fix syslog entries progname to only have the binary name and not the full path (#384)
- Fixed missing home directory on RPM installations (it was unused but caused unnecessary warnings in some cases)
- Fix a deadlock in spooler if the pipeline completely stopped

## 2.6.2

23rd March 2021

Log Carver

- Further improvements and fixes to `if`/`else if`/`else` conditional parsing
- Fix to `grok` action causing events with an empty field name
- Improved configuration error reporting

## 2.6.1

23rd March 2021

Log Courier

- Fixes for panics when loading configuration

Log Carver

- Fixes for `else` conditionals reporting as configuration errors

## 2.6.0

23rd March 2021

Log Courier

- Fix broken `includes` configuration that was broken in 2.5.0 and add preventative tests (#380)
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
- Configuration documentation is minimal but a minimal example can be found in the docs/log-carver/examples folder, more will be added in time

## 2.0.6

9th May 2020

- Fix several issues when shutting down with outstanding payloads, to prevent
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
(<https://github.com/elasticsearch/logstash-forwarder/pull/132> by <https://github.com/atwardowski>)
- Fix duplicated log file import caused by incomplete lines in the log file
(<https://github.com/elasticsearch/logstash-forwarder/pull/164> by <https://github.com/tzahari>)
- Add support for newer SSL certificate types
(<https://github.com/elasticsearch/logstash-forwarder/pull/188> by <https://github.com/pilif>)
- Add support for IPv6 servers
(<https://github.com/elasticsearch/logstash-forwarder/pull/143> by <https://github.com/yath>)
