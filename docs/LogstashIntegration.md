# Logstash Integration

- [Logstash Integration](#logstash-integration)
  - [Overview](#overview)
  - [Installation](#installation)
  - [Configuration](#configuration)

## Overview

Log Courier is built to work seamlessly with [Logstash](http://logstash.net). It
communicates via an input plugin called "courier".

(NOTE: An output plugin exists for Logstash to Logstash transmission but is archived
and no longer maintained and its use is not advised.)

## Installation

Simply run the following commands as the user Logstash was installed with to
install the latest stable version of the Log Courier plugin.

    cd /path/to/logstash
    ./bin/logstash-plugin install logstash-input-courier

Once the installation is complete, you can start using the plugin!

## Configuration

The 'courier' input plugin will now be available. An example configuration follows.

    input {
        courier {
            port            => 12345
            ssl_certificate => "/opt/logstash/ssl/logstash.cer"
            ssl_key         => "/opt/logstash/ssl/logstash.key"
        }
    }

The following options are available:

- transport - "tcp", "tls", "plainzmq" or "zmq" (default: "tls")
- address - Interface address to listen on (defaults to all interfaces)
- port - The port number to listen on
- ssl_certificate - Path to server SSL certificate (tls)
- ssl_key - Path to server SSL private key (tls)
- ssl_key_passphrase - Password for ssl_key (tls, optional)
- ssl_verify - If true, verifies client certificates (tls, default false)
- ssl_verify_default_ca - Accept client certificates signed by systems root CAs
(tls)
- ssl_verify_ca - Path to an SSL CA certificate to use for client certificate
verification (tls)
- min_tls_version - Sets the minimum TLS version when transport is "tls", defaults to 1.2, minimum is 1.0 and maximum 1.3
- max_packet_size - The maximum packet size to accept (default 10485760,
corresponds to Log Courier's `"spool max bytes"`)
- peer_recv_queue - The size of the internal queue for each peer
- add_peer_fields - Add "peer" field to events that identifies source host, and
"peer_ssl_dn" for TLS peers with client certificates
