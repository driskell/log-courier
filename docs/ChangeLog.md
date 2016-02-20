# Change Log

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [1.8.2](#182)
- [1.8.1](#181)
- [1.8](#18)
- [1.7](#17)
- [1.6](#16)
- [1.5](#15)
- [1.3](#13)
- [1.2](#12)
- [1.1](#11)
- [1.0](#10)
- [0.15](#015)
- [0.14](#014)
- [0.13](#013)
- [0.12](#012)
- [0.11](#011)
- [0.10](#010)
- [0.9](#09)
- [Pre-0.10](#pre-010)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## 1.8.3

*???*

***Log Courier***

* Do not open dead files on startup (which causes excessive memory usage if
there are many of them) if their size has not changed since they were last
opened (#242)

***RPM Packaging***

* Fix broken systemd configuration (#237)

***DEB Packaging***

* Fix broken systemd configuration (#237)

## 1.8.2

*30th October 2015*

***Log Courier***

No Changes

***RPM Packaging***

* Fixed incorrect license specification - the spec file specified GPL which was
incorrect - the license is Apache. The spec file has been updated.

***Logstash Plugins***

* Introduce compatibility with Logstash 2.0.0 (#245)

## 1.8.1

*7th August 2015*

***Log Courier***

No Changes

***DEB Packaging***

* Fix a regression in the upstart script due to the addition of a configtest
(#217 - Thanks to @eliecharra)

***Logstash Plugins***

* Fix a regression with `ssl_verify_ca` (#214)
* Fix a regression with the output plugin `hosts` setting (#215)

## 1.8

*6th August 2015*

***Log Courier***

* Fix various causes of multiline codec causing Log Courier to crash (#188)
* Improve handling of file truncation when using codecs (derived from #194)
* Fix "Unknown message received" errors caused by partial reads which occur
frequently with Logstash 1.5.3 due to OpenSSL cipher hardening (#208)
* Fix systemd service configuration providing incorrect arguments to log-courier
(#204)
* Implement configuration test before start and restart for Debian upstart init
script (#189)
* Implement options to enable/disable automatic fields such as "host", "offset"
and "path"
* Implement an option to add a "timezone" field to events containing the local
machine's timezone in the format "-0700 MST" (#203)

***Logstash Plugins***

* Fix broken JSON parser error handling in input plugin (#200)
* Fix Logstash shutdown not working with the input plugin
* Fix broken client certificate verification in the output plugin
* Fix compatibility with Logstash 1.4 due to missing milestone in both plugins

## 1.7

*3rd June 2015*

***Important notice regarding the Logstash Plugins***

The Logstash plugin installation identifier has been changed from
"logstash-xxxxx-log-courier" to "logstash-xxxxx-courier" in order to meet the
specification set by the GA release of Logstash 1.5, which is for the identifier
to match the format "logstash-\<type>-\<configname>". The configuration name is
unchanged and remains "courier" - configurations do not need to be changed.

If you are using Logstash 1.5 and would like to update the plugin you will need
to uninstall the previous plugin using the previous name before installing the
new version using the new name.

    ./bin/plugin uninstall logstash-input-log-courier
    ./bin/plugin uninstall logstash-output-log-courier
    ./bin/plugin install logstash-input-courier
    ./bin/plugin install logstash-output-courier

***Log Courier***

* Report a configuration error if no servers are specified instead of continuing
and crashing during startup (#149)
* Report a configuration error if the configuration file is empty or contains
only whitespace instead of crashing during startup (#148)
* Report a configuration error when a files entry is specified without any paths
instead of continuing and ignoring it (#153)
* Implement server shuffling when multiple servers are specified to prevent
site-wide convergence of Log Courier instances towards a single server (#161)
* Fix pending payload limit being exceeded if a disconnect and reconnect occurs
* Fix a rare race condition in the tcp and tls transport that could cause Log
Courier to stop sending events after a connection failure
* Improve build process to work seamlessly with msysGit on Windows

***Logstash Plugins***

* Rename to logstash-input-courier and logstash-output-courier to meet the
latest plugin specified and fix the "This plugin isn't well supported" warnings
in Logstash 1.5 (#164)
* Remove deprecated milestone declaration to fix a Logstash 1.5 warning (#164)
* Fix a crash in the output plugin that can sometimes occur if a connection
fails (#143)
* Fix a crash on Windows caused by a Logstash patch to Ruby stdlib
(elastic/logstash#3364) (#169)
* Fix confusing log messages that appear to come from Log Courier plugin but
in fact come from another plugin or Logstash itself (#176)
* Fix rare transport failures caused by payload identifiers ending in NUL bytes

## 1.6

*22nd March 2015*

***Plugin-only release***

* The output plugin would fail if the connection to the remote Logstash was lost
and events had to be resent (#136)
* The input and output plugins were incompatible with Logstash RC1 and RC2 due
to breaking changes that occurred after the release of Beta1 (#121)

No changes were made to Log Courier.

## 1.5

*28th February 2015*

***Breaking Changes***

* The way in which logs are read from stdin has been significantly changed. The
"-" path in the configuration is no longer special and no longer reads from
stdin. Instead, you must now start log-courier with the `-stdin` command line
argument, and configure the codec and additional fields in the new `stdin`
configuration file section. Log Courier will now also exit cleanly once all data
from stdin has been read and acknowledged by the server (previously it would
hang forever.)
* The output plugin will fail startup if more than a single `host` address is
provided. Previous versions would simply ignore additional hosts and cause
potential confusion.

***Changes***

* Implement random selection of the initial server connection. This partly
reverts a change made in version 1.2. Subsequent connections due to connection
failures will still round robin.
* Allow use of certificate files containing intermediates within the Log Courier
configuration. (Thanks @mhughes - #88)
* A configuration reload will now reopen log files. (#91)
* Implement support for SRV record server entries (#85)
* Fix Log Courier output plugin (#96 #98)
* Fix Logstash input plugin with zmq transport failing when discarding a message
due to peer_recv_queue being exceeded (#92)
* Fix a TCP transport race condition that could deadlock publisher on a send()
error (#100)
* Fix "address already in use" startup error when admin is enabled on a unix
socket and the unix socket file already exists during startup (#101)
* Report the location in the configuration file of any syntax errors (#102)
* Fix an extremely rare race condition where a dead file may not be resumed if
it is updated at the exact moment it is marked as dead
* Remove use_bigdecimal JrJackson JSON decode option as Logstash does not
support it. Also, using this option enables it globally within Logstash due to
option leakage within the JrJackson gem (#103)
* Fix filter codec not saving offset correctly when dead time reached or stdin
EOF reached (reported in #108)
* Fix Logstash input plugin crash if the fields configuration for Log Courier
specifies a "tags" field that is not an Array, and the input configuration for
Logstash also specified tags (#118)
* Fix a registrar conflict bug that can occur if a followed log file becomes
inaccessible to Log Courier (#122)
* Fix inaccessible log files causing errors to be reported to the Log Courier
log target every 10 seconds. Only a single error should be reported (#119)
* Fix unknown plugin error in Logstash input plugin if a connection fails to
accept (#118)
* Fix Logstash input plugin crash with plainzmq and zmq transports when the
listen address is already in use (Thanks to @mheese - #112)
* Add support for SRV records in the servers configuration (#85)

***Security***

* SSLv2 and SSLv3 are now explicitly disabled in Log Courier and the logstash
courier plugins to further enhance security when using the TLS transport.

## 1.3

*2nd January 2015*

***Changes***

* Added support for Go 1.4
* Added new "host" option to override the "host" field in generated events
(elasticsearch/logstash-forwarder#260)
* The Logstash input gem can now be requested to add extra fields to events for
peer identification. The tls and tcp transports can now add a "peer" field
containing the host and port, and the tls transport a "peer_ssl_cn" field that
will be set to the client certificates common name. The "add_peer_fields" plugin
option will enable these fields (#77)
* Fix missing file in Logstash gem that prevents ZMQ transports from working
(#75)
* Fix Logstash gem crash with ZMQ if a client enters idle state (#73)
* During shutdown, Logstash gem with ZMQ will now never loop with context
terminated errors (#73)
* Significantly improve memory usage when monitoring many files (#78)
* Fix Logstash courier output plugin hanging whilst continuously resending
events and add regression test
* Fix Logstash courier output plugin not verifying the remote certificate
correctly
* Various other minor tweaks and fixes

***Known Issues***

* The Logstash courier output plugin triggers a NameError. This issue is fixed
in the following version. No workaround is available.

## 1.2

*1st December 2014*

***Changes***

* Fix repeated partial Acks triggering an incorrect flush of events to registrar
* Fix a loop that could occur when using ZMQ transport (#68)
* TLS and TCP transport will now round robin the available server addresses
instead of randomising
* Implemented "Dead time in" on `lc-admin` harvester statuses
* `lc-admin` status output is now sorted and no longer in random order
* Add a workaround for logstash shutdown looping with "Context is terminated"
messages (#73)
* Implement asynchronous ZMQ receive pipeline in the Logstash gem to resolve
timeout issues with multiple clients and a busy pipeline
* Implement multithreaded SSL accept in the Logstash gem to prevent a single
hung handshake attempt from blocking new connections
* Switch to ruby-cabin logging in the gems to match Logstash logging
* Updated the RedHat/CentOS 5/6 SysV init script in contrib to follow Fedora
packaging guidelines
* Provided a RedHat/CentOS 7 systemd service configuration in contrib (with
fixes from @matejzero)

***Known Issues***

* The Logstash courier output plugin hangs whilst continuously resending events.
This issue is fixed in the following version. No workaround is available.

## 1.1

*30th October 2014*

* Implement gems for the new Logstash plugin system (#60)
* Fix gem build failing on develop branch with old rubygems versions due to a
malformed version string (#62)
* Fix ZeroMQ transports in the ruby gem with Logstash 1.4.x (#63)
* Fix build issue with ZeroMQ 3.2 and `make with=zmq3`
* Fix partial acknowledgements not being passed to registrar and persisted to
disk
* Fix a race condition when the spooler flushes to prevent a timeout occurring
one or more times after a flush due to size
* Print informational messages containing ZMQ library version information during
gem and log-courier startup to aid in diagnostics
* Raise a friendly error when trying to use the zmq transport in the Log Courier
gem with incompatible versions of libzmq
* Various fixes and improvements to log-courier, gem, build and tests

## 1.0

*23rd October 2014*

* Remove `ping` command from `lc-admin` (#49)
* Empty lines in a log file are incorrectly merged with the following line (#51)
* Don't require a connection to Log Courier when running `lc-admin help` (#50)
* Bring back `make selfsigned` to quickly generate self-signed TLS certificates
(#25)
* Implement `make curvekey` to quickly generate curve key pairs (#25)
* Fix hanging ZMQ transport on transport error
* Fix timeout on log-courier side when Logstash busy due to non-thread safe
timeout timer in the log-courier gem
* Gracefully handle multiline events greater than 10 MiB in size by splitting
events

## 0.15

*23rd September 2014*

* Fix admin being enabled by default when it shouldn't be (#46)

## 0.14

*18th September 2014*

**Breaking Changes**

* The 'file' field in generated events has been renamed to 'path'. This
normalises the events with those generated by Logstash itself, and means the
Logstash `multiline` filter's default `stream_identity` setting is compatible.

**Changes**

* Fix connection failure and retry sometimes entering an unrecoverable state
that freezes log shipping
* Fix ProtocolError with large log packets and on idle connections (since 0.13)
* Provide more information when the gem encounters ProtocolError failures
* Fix ssl_verify usage triggering error, "Either 'ssl_verify_default_ca' or
'ssl_verify_ca' must be specified when ssl_verify is true" (#41)
* Fix previous_timeout multiline codec setting (#45)
* Restore message reliability and correctly perform partial ack. Since 0.9
events from log-courier could be lost after a broken connection and not
retransmitted
* Significantly improve Log Courier gem performance within JRuby by switching
JrJackson parse mode from string to raw+bigdecimal
* Add unix domain socket support to the administration connection
* Provide publisher connection status via the administration connection
* Gracefully handle lines greater than 1 MiB in size by splitting and tagging
them, and make the size configurable (#40)

**Known Issues**

* Admin is enabled by default when it shouldn't be. Workaround: Set the
"admin enabled" general configuration option to false.

## 0.13

*30th August 2014*

* Added new administration utility that can connect to a running Log Courier
instance and report on the current shipping status
* Added new filter codec to allow selective shipping and reduce Logstash loads
* Fixed Logstash plugin entering infinite loop during Logstash shutdown sequence
when using ZMQ. The plugin now shuts down gracefully along with Logstash (#30)
* Fixed unexpected registrar conflict messages appearing for a short time after
a log rotation occurred (#34)
* Fixed Logstash crashing with "Operation cannot be accomplished in current
state" when using ZMQ and Logstash hits a bottleneck requiring partial ACKs to
be sent to Log Courier
* Improved performance of the Log Courier Logstash plugins
* Various other minor rework and improvements

## 0.12

*4th August 2014*

* Fix non-ASCII but valid UTF-8 characters getting replaced with question marks
by the Logstash gem (#22)
* Fix zmq transport not working in Logstash due to ffi-rzmq version too old. Gem
installation will now trigger update of the ffi-rzmq gems to the necessary
versions (#20)
* Fix broken syslog logging (#18)
* Fix broken spool-size configuration setting (#17)
* Fix compilation on Windows (#23)
* Fix shutdown not working when publisher has pending payloads (#24)
* Fix potential race condition issues in the ZMQ Logstash plugin
* Implement ZMQ monitor and log when connections/disconnects happen
* Move logging cmdline settings (such as log-to-syslog) to the configuration
file and allow configuration of stdout logging and file logging as well as
syslog logging (#15)
* Remove support for Go 1.1 due to json.Marshal returning error InvalidUTF8Error
on encountering an invalid sequence. Go 1.2 and above do not and replace invalid
sequence with the Unicode replacement character
* Various other minor tweaks and fixes

## 0.11

*13th July 2014*

* Security fix: Ruby gem client (used by Logstash output plugin) did not verify
 certificate hostname
* Fix edge case file rotation problems (#7)
* Fix incorrect field population in events (#9)
* Fix random hang when Log Courier loses connection to Logstash (#8)
* Improve logging and make the level of detail configurable (#10)
* Fix PING/PONG protocol errors when idle (#12)
* Move spool-size and idle-timeout to the configuration file as "spool size" and
"spool timeout"
* Replace `make selfsigned` with a utility lc-tlscert that can generate
self-signed certificates and the necessary log snippets like genkey did for ZMQ
* Rename genkey to lc-curvekey for consistency
* Various other minor tweaks and fixes

## 0.10

*29th June 2014*

* Support for Go 1.3 (#3)
* Configuration can be reloaded while log courier is running by sending the
SIGHUP signal (\*nix only)
* Additional configuration files can be imported by the main configuration file
using the new `"includes"` section which is an array of Fileglobs. (#5)
* Added `make selfsigned` to allow quick generation of SSL certificates for
testing
* A `"general"` section has been added to the configuration file
* The directory to store the .log-courier persistence file can now be
configured under `"general"/"persist directory"`.
* How often the filesystem is examined for log file appearances or movements
can now be configured under `"general"/"prospect interval"`.
* Fix gem build instructions (#6)
* Fix instances where a file entry has multiple `"fields"` entries results in
all fields having the same value as the first field. (#4)

## 0.9

*14th June 2014*

* Restructure and tidy the project and implement new build tools
* Rename to **Log Courier**
* Implement a completely new test framework with even more tests
* Introduce a new protocol and TLS transport layer that is faster on high
latency links such as the internet
* Implement support for a CurveZMQ transport to allow transmission of logs to
multiple servers simultaneously at low latency.
* Improve efficiency of the event spooler
* Greatly improve the Logstash plugin
* If a log file cannot be opened, only retry as long as the log file exists and
not forever
* Fix offset field on events to point to the start of the event, not the end
* Enable comments inside the configuration file
* Reduce unnecessary logging

## Pre-0.10

The following are fixes present in the Driskell fork of Logstash Forwarder 0.3.1
which Log Courier builds upon.

* Fix state persistence when following multiple files
* Improve log rotation handling
* Make reconnect frequency configurable
* Start from beginning of a log file if created after startup
* Implement partial ACK support into the transmission protocol to prevent
timeout issues and "Too many open files" crashes on remote Logstash instances
* Fix rotation detection on Windows
* Prevent log files from getting locked during harvesting on Windows that
prevents the logging program from renaming the file
* Add a codec system to allow events to be pre-processed before transmission. A
multiline codec to combine multiple lines into single events is now available.
* Fix test suite and add new tests
* Add support for FreeBSD
(https://github.com/elasticsearch/logstash-forwarder/pull/132 by https://github.com/atwardowski)
* Fix duplicated log file import caused by incomplete lines in the log file
(https://github.com/elasticsearch/logstash-forwarder/pull/164 by https://github.com/tzahari)
* Add support for newer SSL certificate types
(https://github.com/elasticsearch/logstash-forwarder/pull/188 by https://github.com/pilif)
* Add support for IPv6 servers
(https://github.com/elasticsearch/logstash-forwarder/pull/143 by https://github.com/yath)
