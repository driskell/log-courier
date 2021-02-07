#!/bin/bash

set -eo pipefail

VERSION=${VERSION#refs/tags/}

echo "::group::Checking exists in $VERSION"
if [ "${NAME}" != "log-courier" ] && [ ! -d "${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

echo "::group::Generating sources for $VERSION"
mkdir -p ~/rpmbuild/{SOURCES,SPECS}
git archive --format=zip --output ~/"rpmbuild/SOURCES/$VERSION.zip" --prefix "log-courier-${VERSION#v}/" "$VERSION"
go mod vendor
zip -qr ~/"rpmbuild/SOURCES/$VERSION.zip" vendor
sed "s/Version: %%VERSION%%/Version: ${VERSION#v}/" <".master/contrib/rpm/${NAME}.spec" >~/"rpmbuild/SPECS/${NAME}.spec"
echo '::endgroup::'

echo '::group::Installing secrets'
mkdir -p ~/.config
cat >~/.config/copr <<EOF
[copr-cli]
copr_url = https://copr.fedorainfracloud.org
EOF
cat >>~/.config/copr <<<"$COPR_CLI"
echo '::endgroup::'

echo '::group::Building SRPM'
yum-builddep -y ~/"rpmbuild/SPECS/${NAME}.spec"
rpmbuild -bs ~/"rpmbuild/SPECS/${NAME}.spec"
echo '::endgroup::'

echo '::group::Testing RPM build'
rpmbuild --rebuild ~/"rpmbuild/SRPMS/${NAME}"-*.src.rpm
echo '::endgroup::'

echo '::group::Submitting to COPR'
copr-cli build log-courier2 ~/"rpmbuild/SRPMS/${NAME}"-*.src.rpm
echo '::endgroup::'
