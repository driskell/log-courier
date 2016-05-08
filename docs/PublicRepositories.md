# Public Repositories

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Redhat / CentOS](#redhat--centos)
- [Ubuntu](#ubuntu)
- [Older Version 1.x Packages](#older-version-1x-packages)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

There are pre-built Log Courier packages readily available in public
repositories for RedHat, CentOS and Ubuntu.

Log Courier is pre-configured to run as a user called `log-courier`, and this
user will need to be given the necessary permissions to access the log files
you wish to ship. Alternatively, Log Courier can be reconfigured to run as any
other user. See the repository description for your platform for notes on how
to do this.

## Redhat / CentOS

*The Log Courier repository depends on the __EPEL__ repository which can be
installed automatically on CentOS distributions by running
`yum install epel-release`. For other distributions, please follow the
installation instructions on the
[EPEL homepage](https://fedoraproject.org/wiki/EPEL).*

To install the Log Courier YUM repository, download the corresponding `.repo`
configuration file below, and place it in `/etc/yum.repos.d`. Log Courier may
then be installed using `yum install log-courier`.

* **CentOS/RedHat 6.x**: [driskell-log-courier2-epel-6.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier2/repo/epel-6/driskell-log-courier2-epel-6.repo)
* **CentOS/RedHat 7.x**:
[driskell-log-courier2-epel-7.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier2/repo/epel-7/driskell-log-courier2-epel-7.repo)

Once installed, create a configuration file at
`/etc/log-courier/log-courier.yaml` to suit your needs. If you need to change
the user that Log Courier runs as, edit `/etc/sysconfig/log-courier`. Then when
you're all set, start the Log Courier service to begin shipping.

    service log-courier start

## Ubuntu

To install the Log Courier apt-get repository, run the following commands.

    sudo add-apt-repository ppa:devel-k/log-courier2
    sudo apt-get update

Log Courier may then be installed using `apt-get install log-courier`.

Once installed, create a configuration file at
`/etc/log-courier/log-courier.yaml` to suit your needs. If you need to change
the user that Log Courier runs as, edit `/etc/default/log-courier`. Then when
you're all set, start the Log Courier service to begin shipping.

    service log-courier start

**NOTE:** The Ubuntu packages have had limited testing and you are welcome to
give feedback and raise feature requests or bug reports to help improve them!

**NOTE:** Packages for Ubuntu `precise` packages are no longer available due to
limitations on the available Golang versions. However, the old 1.x packages will
still continue to be available for Ubuntu `precise`.

## Older Version 1.x Packages

If you still require the older version 1.x packages, use the following
configuration files for CentOS or RedHat.

* **CentOS/RedHat 6.x**: [driskell-log-courier-epel-6.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-6/driskell-log-courier-epel-6.repo)
* **CentOS/RedHat 7.x**:
[driskell-log-courier-epel-7.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier/repo/epel-7/driskell-log-courier-epel-7.repo)

For Ubuntu, use `ppa:devel-k/log-courier` instead of `ppa:devel-k/log-courier2`.

In contract to the current packages, the version 1.x packages refer to a JSON
configuration file at `/etc/log-courier/log-courier.conf` and always run as the
`root` user.
