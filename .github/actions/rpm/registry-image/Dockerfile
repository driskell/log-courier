FROM almalinux:9
RUN dnf -y install rpm-sign rpm-build epel-release ca-certificates
RUN dnf -y install copr-cli golang git
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
