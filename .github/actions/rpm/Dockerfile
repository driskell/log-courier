FROM centos:7
RUN yum -y install git rpm-sign rpm-build epel-release ca-certificates
RUN yum -y install copr-cli golang
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
