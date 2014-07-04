# Logstash Integration

Log Courier is built to work seamlessly with [Logstash](http://logstash.net)
1.4.x.

## Installation

To enable communication with Logstash the ruby gem needs to be installed into
the Logstash installation, and then the input and output plugins.

The following instructions assume you are using the tar.gz or packaged Logstash
installations and that Logstash is installed to /opt/logstash. You should change
this path if yours is different.

First build the gem. This will generate a file called log-courier-X.X.gem.

		git clone https://github.com/driskell/log-courier
		cd log-courier
		gem build log-courier.gemspec

Then switch to the Logstash installation directory and install it. Note that
because this is JRuby it may take a minute to finish the install.

		cd /opt/logstash
		export GEM_HOME=vendor/bundle/jruby/1.9
		java -jar vendor/jar/jruby-complete-1.7.11.jar -S gem install <path-to-gem>

Now install the Logstash plugins.

		cd <log-courier-source>
		cp -rvf lib/logstash /opt/logstash/lib

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

* address - Interface address to listen on (defaults to all interfaces)
* transport - "tcp", "tls" (default), "plainzmq" or "zmq"
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

NOTE: The tcp, plainzmq and zmq transports are not implemented in the output
plugin at this time. It supports only the tls transport.
