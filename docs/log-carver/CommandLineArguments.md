# Command Line Arguments

- [Command Line Arguments](#command-line-arguments)
  - [Overview](#overview)
  - [`-config=<path>`](#-configpath)
  - [`-config-test`](#-config-test)
  - [`-cpuprofile=<path>`](#-cpuprofilepath)
  - [`-list-supported`](#-list-supported)
  - [`-version`](#-version)

## Overview

The `log-carver` command accepts various command line arguments.

## `-config=<path>`

The path to the JSON configuration file to load.

```shell
log-carver -config=/etc/log-carver/log-carver.json
```

## `-config-test`

Load the configuration and test it for validity, then exit.

Will exit with code 0 if the configuration is valid and would not prevent
log-carver from starting up. Will exit with code 1 if an error occurred,
printing the error to standard output.

## `-cpuprofile=<path>`

The path to file to write CPU profiling information to, when investigating
performance problems. Log Carver will run for a small period of time and then
quit, writing the profiling information to this file.

This flag should generally only be used when requested by a developer.

## `-list-supported`

Print a list of available transports, receivers and actions provided by this build of Log Carver, then exit.

## `-version`

Print the version of this build of Log Carver, then exit.
