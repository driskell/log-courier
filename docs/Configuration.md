# Log Courier Configuration

## Overview

The Log Courier configuration is currently stored in standard JSON format with
the exception that comments are allowed. It is split into two sections,
network and files.

```
	{
		"network": {
			# Network configuration here
		},
		"files": {
			# File configuration here
		}
	}
```

## "network"

The network configuration tells Log Courier where to ship the logs, and also
what transport and security to use.

### "transport"

*String. Optional. Default: "tls"*  
Available values: "tls", "zmq"

Sets the transport to use when sending logs to the servers. "tls" is recommended
for most users and connects to a single server at random, reconnecting to a
different server at random each time the connection fails. "zmq" connects to all
specified servers and load balances events across them.

"zmq" is only available if Log Courier was compiled with the "with=zmq" option, which
requires ZeroMQ >= 4.0.0 to be installed.

### "servers"

*Array of strings. Required*

Sets the list of servers to send logs to. DNS names are resolved into IP
addresses each time connections are made and all available IP addresses are
used.

### "ssl ca"

*Filepath. Required*

### "ssl certificate"

*Filepath. Optional*

### "ssl key"

*Filepath. Required with "ssl certificate"*

### "timeout"

*Duration. Optional. Default: 5*

### "reconnect"

*Duration. Optional. Default: 1*

## "files"

The file configuration lists the file groups that contain the logs you wish to
ship. It is an array of group configurations. A minimum of one file group
configuration must be specified.

```
	[
		{
			# First file group
		},
		{
			# Second file group
		}
	]
```

### "paths"

*Array of filepaths. Required*

### "fields"

*Dictionary. Optional*

### "dead time"

*Duration. Optional. Default: "24h"

### "codec"

*Codec configuration. Optional  
Default: Plain codec*

## Examples

Several configuration examples are available for you perusal.

1. Ship a single log file
2. Ship a folder of log files
3. Ship logs with extra field information
4. Multiline log processing
5. Using ZMQ to load balance
