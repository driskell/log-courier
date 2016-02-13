# Logstash Integration

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Installation](#installation)
  - [Logstash Plugin Manager](#logstash-plugin-manager)
  - [Manual installation](#manual-installation)
  - [Local-only Installation](#local-only-installation)
- [Configuration](#configuration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

Log Courier is built to work seamlessly with [Logstash](http://logstash.net). It
communicates via an input plugin called "courier".

An output plugin is also available to allow Logstash instances to communicate
with each other using the same reliable and efficient protocol as Log Courier.

## Installation

### Logstash Plugin Manager

Logstash 1.5 introduces a new plugin manager that makes installing additional
plugins extremely easy.

Simply run the following commands as the user Logstash was installed with to
install the latest stable version of the Log Courier plugins. If you are only
receiving events, you only need to install the input plugin.

    cd /path/to/logstash
    bin/plugin install logstash-input-courier
    bin/plugin install logstash-output-courier

Once the installation is complete, you can start using the plugins!

*Note: If you receive a Plugin Conflict error, try updating the zeromq output
plugin first using `bin/plugin update logstash-output-zeromq`*

### Manual installation

For Logstash 1.4.x the plugins and dependencies need to be installed manually.

First build the Log Courier gem the plugins require. The file you will need will
be called log-courier-X.X.gem, where X.X is the version of Log Courier you have.

    git clone https://github.com/driskell/log-courier
    cd log-courier
    make gem

Switch to the Logstash installation directory as the user Logstash was installed
with and install the gem. Note that because this is JRuby it may take a minute
to finish the install. The ffi-rzmq-core and ffi-rzmq gems bundled with Logstash
will be upgraded during the installation, which will require an internet
connection.

    cd /path/to/logstash
    export GEM_HOME=vendor/bundle/jruby/1.9
    java -jar vendor/jar/jruby-complete-1.7.11.jar -S gem install /path/to/the.gem

The remaining step is to manually install the Logstash plugins.

    cd /path/to/log-courier
    cp -rvf lib/logstash /path/to/logstash/lib

### Local-only Installation

If you need to install the gem and plugins on a server without an internet
connection, you can download the gem dependencies from the rubygems site and
transfer them across. Follow the instructions for Manual Installation and
install the dependency gems first using the same instructions as for the Log
Courier gem.

* https://rubygems.org/gems/ffi-rzmq-core
* https://rubygems.org/gems/ffi-rzmq
* https://rubygems.org/gems/multi_json

## Configuration

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

* transport - "tcp", "tls", "plainzmq" or "zmq" (default: "tls")
* address - Interface address to listen on (defaults to all interfaces)
* port - The port number to listen on
* ssl_certificate - Path to server SSL certificate (tls)
* ssl_key - Path to server SSL private key (tls)
* ssl_key_passphrase - Password for ssl_key (tls, optional)
* ssl_verify - If true, verifies client certificates (tls, default false)
* ssl_verify_default_ca - Accept client certificates signed by systems root CAs
(tls)
* ssl_verify_ca - Path to an SSL CA certificate to use for client certificate
verification (tls)
* curve_secret_key - CurveZMQ secret key for the server (zmq)
* max_packet_size - The maximum packet size to accept (default 10485760,
corresponds to Log Courier's `"spool max bytes"`)
* peer_recv_queue - The size of the internal queue for each peer
* add_peer_fields - Add "peer" field to events that identifies source host, and
"peer_ssl_dn" for TLS peers with client certificates

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
* idle_timeout - Maximum time in seconds to wait for a full spool before
flushing anyway (default 5)

NOTE: The tcp, plainzmq and zmq transports are not implemented in the output
plugin at this time. It supports only the tls transport.
