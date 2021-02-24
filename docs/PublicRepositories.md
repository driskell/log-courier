# Public Repositories

- [Public Repositories](#public-repositories)
  - [Overview](#overview)
  - [Redhat / CentOS](#redhat--centos)
  - [Ubuntu](#ubuntu)
  - [Package Names](#package-names)

## Overview

There are pre-built packages readily available in public repositories for RedHat, CentOS and Ubuntu.

Log Courier is pre-configured to run as a user called `log-courier`, and this user will need to be given the necessary permissions to access the log files you wish to ship. Alternatively, Log Courier can be reconfigured to run as any other user. See the repository description for your platform for notes on how to do this.

Log Carver is pre-configured to run as a user called `log-carver`. This generally does not need changing as Log Carver will receive events over the network and then store them in Elasticsearch.

## Redhat / CentOS

*The Log Courier repository depends on the __EPEL__ repository which can be installed automatically on CentOS distributions by running `yum install epel-release`. For other distributions, please follow the installation instructions on the [EPEL homepage](https://fedoraproject.org/wiki/EPEL).*

To install the Log Courier YUM repository, download the corresponding `.repo` configuration file below, and place it in `/etc/yum.repos.d`.

- **CentOS Stream**:
[driskell-log-courier2-centos-stream.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier2/repo/centos-stream/driskell-log-courier2-centos-stream.repo)
- **EPEL for CentOS 8.x / RedHat 8.x**:
[driskell-log-courier2-epel-8.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier2/repo/epel-8/driskell-log-courier2-epel-8.repo)
- **EPEL for CentOS 7.x / RedHat 7.x**:
[driskell-log-courier2-epel-7.repo](https://copr.fedoraproject.org/coprs/driskell/log-courier2/repo/epel-7/driskell-log-courier2-epel-7.repo)

## Ubuntu

To install the apt-get repository, run the following commands.

    sudo add-apt-repository ppa:devel-k/log-courier2
    sudo apt-get update

**NOTE:** The Ubuntu packages have had limited testing and you are welcome to
give feedback and raise feature requests or bug reports to help improve them!

## Package Names

The packages available in the above repositories are as follows:

- log-courier
- log-carver
