# Change Log

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
