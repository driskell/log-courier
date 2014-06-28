# Log Courier [![Build Status](https://travis-ci.org/driskell/log-courier.svg?branch=stable)](https://travis-ci.org/driskell/log-courier)

Log Courier is a tool created to transmit log files speedily and securely to
remote [Logstash](http://logstash.net) instances for processing whilst using
small amounts of local resources. The project is an enhanced fork of
[Logstash Forwarder](https://github.com/elasticsearch/logstash-forwarder) 0.3.1
with many enhancements and behavioural improvements.

## Features

Log Courier implements the following features:

* Tail log files, following rotations and resuming at the last offset on
restart
* Read from standard input for lightweight shipping of a program's output
* Extra event fields, arrays and hashes on a per file basis
* Fast and secure transmission of logs using TLS with both server and client
certificate verification
* Multiline codec to combine multiple lines into single events prior to shipping
* A ruby gem to enable fast and secure transmission of logs between Logstash
instances
* Transmission of logs via CurveZMQ to multiple receivers simultaneously
(optional, requires ZeroMQ 4+)

## Installation

### Build Requirements

1. The [go](http://golang.org/doc/install) compiler tools (>= 1.1.0)
1. [Logstash](http://logstash.net) 1.4.x
1. (Optional) [ZeroMQ](http://zeromq.org/intro:get-the-software) (>= 4.0.0)

### Building

To build with the optional ZMQ support use the following.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make with=zmq

Otherwise, simply run make standalone as follows.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make

(If you receive errors, try using gmake instead.)

The log-courier program can then be found in the 'bin' folder.

A genkey utility can also be found in the 'bin' folder when ZMQ support is
built. This utility will generate CurveZMQ key pair configurations for you.

## Command Line Options

The log-courier command accepts the following command line options.

    -config="": The config file to load
    -cpuprofile="": write cpu profile to file
    -from-beginning=false: Read new files from the beginning, instead of the end
    -idle-flush-time=5s: Maximum time to wait for a full spool before flushing anyway
    -log-to-syslog=false: Log to syslog instead of stdout
    -spool-size=1024: Maximum number of events to spool before a flush is forced

## Documentation

* [Logstash Integration](docs/LogstashIntegration.md)
* [Configuration](docs/Configuration.md)
* [Change Log](docs/ChangeLog.md)
