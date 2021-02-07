FROM ubuntu:focal
RUN apt -qy update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qy install git devscripts debhelper sbuild ubuntu-dev-tools golang
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
