#!/bin/bash

set -eo pipefail

VERSION=${VERSION#refs/tags/}

echo "::group::Checking exists in $VERSION"
if [ "${NAME}" != "log-courier" ] && [ ! -d "${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

echo "::group::Generating sources for $REF"
mkdir -p ~/rpmbuild/{SOURCES,SPECS}
git archive --format=zip --output ~/"rpmbuild/SOURCES/$VERSION.zip" --prefix "log-courier-${VERSION#v}/" "$REF"
echo '::endgroup::'

echo "::group::Adding vendored modules to source"
go mod vendor
# Clear cache after vendoring, so that the subsequent test RPM build does not try to use a VCS cache
# This will allow us to detect vendoring issues as we will then see it attempt to download additional items
go clean -cache -modcache -i -r
ln -nsf . "log-courier-${VERSION#v}"
zip -qr ~/"rpmbuild/SOURCES/$VERSION.zip" "log-courier-${VERSION#v}/vendor"
echo '::endgroup::'

echo "::group::Generating spec for $VERSION"
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
copr-cli build "${TARGET_REPO}" ~/"rpmbuild/SRPMS/${NAME}"-*.src.rpm
echo '::endgroup::'
