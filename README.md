# Log Courier [[!Build Status](https://travis-ci.org/driskell/log-courier.svg)](https://travis-ci.org/driskell/log-courier)

Log Courier is a tool created to transmit log files speedily and securely to
remote [LogStash](http://logstash.net) instances for processing whilst using
small amounts of local resources. The project is an enhanced fork of the
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder)
project with many enhancements and behavioural improvements.

## Features

Log Courier implements the following features:

* Tail text log files, following rotations and resuming at the last offset on
restart
* Listen to standard input at the end of a shell pipeline
* Extra fields can be tagged onto events on a per file basis containing simple
strings and numbers or deep arrays and hashes
* Fast and secure transmission of logs using TLS with both server and client
certificate verification
* Transmission of logs via CurveZMQ to multiple LogStash receivers
simultaneously (optional, requires ZeroMQ 4+)
* Multiline codec to simplify the LogStash configuration and improve indexing
speeds
* Multiline timeout to ensure events are transmitted without waiting for the
next
* Fast and secure transmission of logs between LogStash instances over TLS using
the log-courier ruby gem

## Installation

### Requirements

1. The [go](http://golang.org/doc/install) compiler tools (>= 1.1.0)
2. (Optional) [ZeroMQ](http://zeromq.org/intro:get-the-software) (>= 4.0.0)

### Building

To build with the optional ZMQ support, use the following.

    make with=zmq

Otherwise, simply run make standalone, as follows.

    make

## Configuration

Before you can start log-courier you will need to create a configuration file.

Configuration is in standard JSON format with the exception that comments are
allowed. A pound sign (#) designates a comment until the end of that line.
A multiline comment can also be started using /* and ended using */.

Full documentation of all of the available configuration options can be found on
the [Configuration page](docs/Configuration.md). A brief example is shown below
to get you started.

    /*
      This is a brief example configuration
      for log-courier to get you started
    */
    {
      "network": {
        "servers": [ "logstash.example.com:5000" ],
        "ssl certificate": "./courier.cer",
        "ssl key": "./courier.key",
        "ssl ca": "./logstash.cer"
      },
      "files": [
        {
          "paths": [
            "/var/log/*.log",
            "/var/log/messages"
          ],
          "fields": { "type": "syslog", "tags": [ "system", "dba" ] }
        }, {
          "paths": [ "/var/log/httpd/access.log" ],
          "fields": { "type": "apache" }
        }
      ]
    }

## Command Line Options

    -config="": The config file to load
    -cpuprofile="": write cpu profile to file
    -from-beginning=false: Read new files from the beginning, instead of the end
    -idle-flush-time=5s: Maximum time to wait for a full spool before flushing
    anyway
    -log-to-syslog=false: Log to syslog instead of stdout
    -spool-size=1024: Maximum number of events to spool before a flush is forced
