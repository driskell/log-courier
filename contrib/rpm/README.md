Packaging Instructions
======================

Install the required package building packages.

```
sudo yum install rpm-build golang
```

Setup a standard RPM build workspace with the required folders.

```
mkdir -p rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
```

Run the following commands to download the required source archive, the RPM spec
file and build the package.

```
export VERSION=<your version here> # set to the version you wish to download and build
wget -P rpmbuild/SOURCES https://github.com/driskell/log-courier/archive/v${VERSION}.zip
wget -P rpmbuild/SPECS https://raw.githubusercontent.com/driskell/log-courier/v${VERSION}/contrib/rpm/log-courier.spec
sed -i "s/^Version: .*$/Version: ${VERSION}/" rpmbuild/SOURCES/log-courier.spec
rpmbuild -ba rpmbuild/SPECS/log-courier.spec
```

Building a Source RPM
=====================

Follow the instructions for a binary package using the following rpmbuild
parameters instead.

```
rpmbuild -bs --sign rpmbuild/SPECS/log-courier.spec
```
