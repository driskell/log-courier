# Change Log

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](http://doctoc.herokuapp.com/)*

- [0.13](#013)
- [0.12](#012)
- [0.11](#011)
- [0.10](#010)
- [0.9](#09)
- [Pre-release](#pre-release)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## ?.??

*???*

* Fix hang that sometimes occurs during connection retries
* Fix ProtocolError with large log packets and on idle connections (since 0.13)
* Provide more information when the gem encounters ProtocolError failures
* Fix ssl_verify usage triggering error, "Either 'ssl_verify_default_ca' or
'ssl_verify_ca' must be specified when ssl_verify is true"
* Restore message reliability and correctly perform partial ack. Since 0.9 a
full event spool from log-courier could be lose (default 1024) and partial ack
would only be performed if 0 events could be queued
* Significantly improve Log Courier gem performance within JRuby by switching
JrJackson parse mode from string to raw+bigdecimal

## 0.13

*30th August 2014*

* Added new administration utility that can connect to a running Log Courier
instance and report on the current shipping status
* Added new filter codec to allow selective shipping and reduce LogStash loads
* Fixed LogStash plugin entering infinite loop during LogStash shutdown sequence
when using ZMQ. The plugin now shuts down gracefully along with LogStash (#30)
* Fixed unexpected registrar conflict messages appearing for a short time after
a log rotation occurred (#34)
* Fixed LogStash crashing with "Operation cannot be accomplished in current
state" when using ZMQ and LogStash hits a bottleneck requiring partial ACKs to
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
SIGHUP signal (*nix only)
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
* Implement an completely new test framework with even more tests
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

## Pre-release

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
