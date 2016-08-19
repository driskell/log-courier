Packaging Instructions
======================

Install required package building packages.

    sudo apt-get install devscripts debhelper golang-go

Run the following commands to get the source, create the required upstream
archive, setup the source for debian building, and build the package.

    git clone https://github.com/driskell/log-courier
    tar -czf log-courier_VERSION.orig.tar.gz log-courier
    cd log-courier
    # You can build the package for either upstart or systemd
    # upstart
    mv contrib/deb-upstart debian
    # systemd
    # mv contrib/deb-systemd debian
    dpkg-buildpackage

Packaging on Wheezy
===================

The version of 'go' available in the golang-go package available in Debian
wheezy is not recent enough to build this package. The following procedure
can be followed as a workaround.

First, grab the latest go from https://golang.org/ and install as standard in
/usr/local/go (debian's current packaged version is not new enough).

Add go to the PATH

    export PATH=/usr/local/go/bin:$PATH

Build the packages as normal.

Building a Source DEB
=====================

Follow the instructions for a regular package using the following command
instead of dpkg-buildpackage.

    debuild -S -sa

If you need to set the target distribution for the source package, such as for
upload to PPA, you can run the following command before debuild to set it,
replacing trusty with the desired distribution name.

    dch -r -M -D trusty
