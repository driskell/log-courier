# ZeroMQ Support

*NOTICE: ZeroMQ support will no longer be available as of Log Courier version
2.0, where the 'tcp' and 'tls' transports will be able to connect to multiple
Logstash instances. This reduces the benefits of retaining ZeroMQ support
significantly, especially as the new 'tcp' and 'tls' implementation will recover
from instance failures significantly faster. Additionally, the library currently
in use has entered maintenance only support and as such would need replacing to
allow it to continue to take advantage of future ZeroMQ improvements.*

The ZeroMQ transports allow Log Courier to connect to multiple Logstash
instances simultaenously and load balance events between them in an efficient
and reliable manner.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Packaged Versions of Log Courier](#packaged-versions-of-log-courier)
- [Building from Source](#building-from-source)
- [Generating Keys for the 'zmq' Transport](#generating-keys-for-the-zmq-transport)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Packaged Versions of Log Courier

All current packages are built with support for the cleartext 'plainzmq'
transport.

Because most distrbutions do not provide packaged versions of ZeroMQ >=4.0,
it is not possible to provide a packaged version of Log Courier that supports
the 'zmq' transport, as this transport requires ZeroMQ >=4.0.

## Building from Source

To build with support for the 'plainzmq' or 'zmq' transports, you will need to
install [ZeroMQ](http://zeromq.org/intro:get-the-software) (>=3.2 for
'plainzmq', >=4.0 for 'zmq' which supports encryption).

***Linux\Unix:*** *ZeroMQ >=3.2 is usually available via the package manager.
ZeroMQ >=4.0 may need to be built and installed manually.*  
***OS X:*** *ZeroMQ can be installed via [Homebrew](http://brew.sh).*  
***Windows:*** *ZeroMQ will need to be built and installed manually.*

Once the required version of ZeroMQ is installed, run the corresponding `make`
command to build Log Courier with the ZMQ transports.

    # ZeroMQ >=3.2 - cleartext 'plainzmq' transport
    make with=zmq3
    # ZeroMQ >=4.0 - both cleartext 'plainzmq' and encrypted 'zmq' transport
    make with=zmq4

*Note: If you receive errors whilst running `make`, try `gmake` instead.*

**Please ensure that the versions of ZeroMQ installed on the Logstash hosts and
the Log Courier hosts are of the same major version. A Log Courier host that has
ZeroMQ 4.0.5 will not work with a Logstash host using ZeroMQ 3.2.4 (but will
work with a Logstash host using ZeroMQ 4.0.4.)**

## Generating Keys for the 'zmq' Transport

Log Courier provides a command to help generate Curve keys: `lc-curvekey`.

Running `make curvekey` will automatically build and run the `lc-curvekey`
utility that will quickly and easily generate Curve key pairs, along with the
corresponding configuration snippets, for the 'zmq' shipping transport.
