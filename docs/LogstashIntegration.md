# Logstash Integration

- [Logstash Integration](#logstash-integration)
  - [Overview](#overview)
  - [Installation](#installation)
  - [Input Configuration](#input-configuration)
  - [Output Configuration](#output-configuration)

## Overview

Log Courier can be used to send events to [Logstash](http://logstash.net) by installing an input plugin.

Additionally, Logstash can send events to Log Carver by installing an output plugin.

## Installation

Simply run the following commands as the user Logstash was installed with to install the latest stable version of the Log Courier input plugin.

    cd /path/to/logstash
    ./bin/logstash-plugin install logstash-input-courier

To install the output plugin, run the following

    cd /path/to/logstash
    ./bin/logstash-plugin install logstash-output-courier

## Input Configuration

An example configuration for the `courier` input plugin is below:

    input {
        courier {
            port            => 12345
            ssl_certificate => "/opt/logstash/ssl/logstash.cer"
            ssl_key         => "/opt/logstash/ssl/logstash.key"
        }
    }

The following options are available:

- transport - "tcp" or "tls" (default: "tls")
- address - Interface address to listen on (defaults to all interfaces)
- port - The port number to listen on
- ssl_certificate - Path to server SSL certificate (tls)
- ssl_key - Path to server SSL private key (tls)
- ssl_key_passphrase - Password for ssl_key (tls, optional)
- ssl_verify - If true, verifies client certificates (tls, default false)
- ssl_verify_default_ca - Accept client certificates signed by systems root CAs (tls)
- ssl_verify_ca - Path to an SSL CA certificate to use for client certificate verification (tls)
- min_tls_version - Sets the minimum TLS version when transport is "tls", defaults to 1.2, minimum is 1.0 and maximum 1.3
- max_packet_size - The maximum packet size to accept (default 10485760, corresponds to Log Courier's `spool max bytes`)
- add_peer_fields - Add "peer" field to events that identifies source host, and "peer_ssl_dn" for TLS peers with client certificates

## Output Configuration

An example configuration for the `courier` output plugin is below:

    output {
        courier {
            addresses       => ['127.0.0.1']
            port            => 12345
            ssl_certificate => "/opt/logstash/ssl/logstash.cer"
        }
    }

The following options are available:

- transport - "tcp" or "tls" (default: "tls")
- addresses - Address to connect to in array format (only one value is supported at the moment)
- port - Port to connect to
- ssl_ca - Path to SSL certificate to verify server certificate
- ssl_certificate - Path to client SSL certificate (optional)
- ssl_key - Path to client SSL private key (optional)
- ssl_key_passphrase - Password for ssl_key (optional)
- min_tls_version - Sets the minimum TLS version when transport is "tls", defaults to 1.2, minimum is 1.0 and maximum 1.3
- spool_size - Maximum number of events to spool before a flush is forced (default 1024)
- idle_timeout - Maximum time in seconds to wait for a full spool before flushing anyway (default 5)
