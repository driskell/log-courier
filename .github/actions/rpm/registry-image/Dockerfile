FROM centos:7
RUN yum -y install rpm-sign rpm-build epel-release ca-certificates centos-release-scl
RUN yum -y install copr-cli golang rh-git218
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
