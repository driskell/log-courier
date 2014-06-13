# Log Courier [![Build Status](https://travis-ci.org/driskell/log-courier.svg)](https://travis-ci.org/driskell/log-courier)

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

### Build Requirements

1. The [go](http://golang.org/doc/install) compiler tools (>= 1.1.0)
2. (Optional) [ZeroMQ](http://zeromq.org/intro:get-the-software) (>= 4.0.0)

### Building

To build with the optional ZMQ support, use the following.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make with=zmq

Otherwise, simply run make standalone, as follows.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make

The log-courier program can then be found in the 'bin' folder.

A genkey utility can also be found in 'bin' when ZMQ support is built. This
utility will generate CurveZMQ key pair configurations for you.

### LogStash 1.4.x Integration

To enable communication with LogStash the ruby gem needs to be installed into
the LogStash installation, and then the input and output plugins.

The following instructions assume you are using the tar.gz or packaged LogStash
installations and that LogStash is installed to /opt/logstash. You should change
this path if yours is different.

First build the gem. This will generate a file called log-courier-X.X.gem.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    gem build

Then switch to the LogStash installation directory and install it. Note that
because this is JRuby it may take a minute to finish the install.

    cd /opt/logstash
    export GEM_HOME=vendor/bundle/jruby/1.9
    java -jar vendor/jar/jruby-complete-1.7.11.jar -S gem install <path-to-gem>

Now install the LogStash plugins.

    cd <log-courier-source>
    cp -rvf lib/logstash /opt/logstash/lib

The 'courier' input and output plugins will now be available. An example
configuration for the input plugin follows.

    input {
        courier {
            port            => 12345
            ssl_certificate => "/opt/logstash/ssl/logstash.cer"
            ssl_key         => "/opt/logstash/ssl/logstash.key"
        }
    }

The following options are available for the input plugin:

* address - Interface address to listen on (defaults to all interfaces)
* transport - "tls" (default) or "zmq"
* ssl_certificate - Path to server SSL certificate
* ssl_key - Path to server SSL private key
* ssl_key_passphrase - Password for ssl_key (optional)
* ssl_verify - If true, verifies client certificates (default false)
* ssl_verify_default_ca - Accept client certificates signed by systems root CAs
* ssl_verify_ca - Path to an SSL CA certificate to use for client certificate
verification

The following options are available for the output plugin:

* addresses - Address to connect to in array format (only the first address will
be used at the moment)
* port - Port to connect to
* ssl_ca - Path to SSL certificate to verify server certificate
* ssl_certificate - Path to client SSL certificate (optional)
* ssl_key - Path to client SSL private key (optional)
* ssl_key_passphrase - Password for ssl_key (optional)
* spool_size - Maximum number of events to spool before a flush is forced
(default 1024)
* idle_timeout - Maxmimum time in seconds to wait for a full spool before
flushing anyway (default 5)

NOTE: The ZMQ transport is not implemented in the output plugin at this time.

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
