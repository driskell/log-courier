FROM ubuntu:focal
RUN apt -qy update
RUN DEBIAN_FRONTEND=noninteractive apt-get -qy install git devscripts debhelper sbuild ubuntu-dev-tools
RUN echo '1;' > /root/.sbuildrc
RUN gpasswd -a root sbuild
RUN sed -i '1179s%umount%#umount%' /usr/share/debootstrap/functions
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
