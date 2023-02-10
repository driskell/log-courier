FROM ubuntu:jammy
RUN apt -qy update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qy install git devscripts debhelper sbuild ubuntu-dev-tools golang ca-certificates
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
