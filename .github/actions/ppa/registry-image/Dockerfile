# Note that we cross-compile in this image
# Therefore the builds this image produces for upload to PPA must be compatibile with all our targets
# This means we must compile on the oldest version of Ubuntu we support, which is trusty
# However, we need to support systemd (since xenial)
# The earliest crossover therefore is bionic - which fortunately uses similar GLIBC that means binaries also work on trusty and xenial
FROM ubuntu:bionic
RUN apt -qy update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qy install git devscripts debhelper sbuild ubuntu-dev-tools ca-certificates
RUN wget -q https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -O /tmp/go.tar.gz
RUN tar -C /usr/local -xzf /tmp/go.tar.gz
RUN rm -f /tmp/go.tar.gz
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
