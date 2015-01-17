# Command Line Arguments

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](http://doctoc.herokuapp.com/)*

- [Overview](#overview)
- [`-config=<path>`](#-config=path)
- [`-config-test`](#-config-test)
- [`-cpuprofile=<path>`](#-cpuprofile=path)
- [`-from-beginning`](#-from-beginning)
- [`-list-supported`](#-list-supported)
- [`-stdin`](#-stdin)
- [`-version`](#-version)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The `log-courier` command accepts various command line arguments.

## `-config=<path>`

The path to the JSON configuration file to load.

```
log-courier -config=/etc/log-courier/log-courier.json
```

## `-config-test`

Load the configuration and test it for validity, then exit.

Will exit with code 0 if the configuration is valid and would not prevent
log-courier from starting up. Will exit with code 1 if an error occurred,
printing the error to standard output.

## `-cpuprofile=<path>`

The path to file to write CPU profiling information to, when investigating
performance problems. Log Courier will run for a small period of time and then
quit, writing the profiling information to this file.

This flag should generally only be used when requested by a developer.

## `-from-beginning`

The `.log-courier` file stores the current shipping status as logs are shipped
so that in the event of a service restart, not a single log entry is missed.

In the event that the `.log-courier` file does not exist, Log Courier will by
default start the initial shipping of log files from the end of the file.
Setting this flag in the initial startup of Log Courier will trigger files to
start shipping from the beginning of the file instead of the end.

After the first `.log-courier` status file is written, all subsequent newly
discovered log files will start from the begining, regardless of this flag.

## `-list-supported`

Print a list of available transports and codecs provided by this build of Log
Courier, then exit.

## `-stdin`

Read log data from stdin and ignore files declaractions in the configuration
file. The fields and codec can be configured in the configuration file under
the `"stdin"` section.

## `-version`

Print the version of this build of Log Courier, then exit.
