FROM centos:7
RUN yum -y install git rpm-sign rpm-build epel-release
RUN yum -y install copr-cli
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
