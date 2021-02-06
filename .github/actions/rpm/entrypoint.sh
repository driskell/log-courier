#!/bin/bash

set -eo pipefail

echo "::group::Checking exists in $VERSION"
if [ "${NAME}" != "log-courier" ] && [ ! -d "/github/workspace/${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

echo "::group::Generating sources for $VERSION"
mkdir -p ~/rpmbuild/{SOURCES,SPECS}
GIT_DIR=/github/workspace/.git git archive --format=zip --output ~/"rpmbuild/SOURCES/$VERSION.zip" --prefix "${NAME}-${VERSION#v}/" "$VERSION"
cp -f "contrib/rpm/${NAME}.spec" ~/rpmbuild/SPECS
sed "s/Version: %%VERSION%%/Version: ${VERSION#v}/" <"contrib/rpm/${NAME}.spec" >~/"rpmbuild/SPECS/${NAME}.spec"
echo '::endgroup::'

echo '::group::Installing secrets'
mkdir -p ~/.config
cat >~/.config/copr <<EOF
[copr-cli]
copr_url = https://copr.fedorainfracloud.org
EOF
cat >>~/.config/copr <<<"$COPR_CLI"
cat ~/.config/copr
echo '::endgroup::'

echo '::group::Building SRPM'
yum-builddep -y ~/"rpmbuild/SPECS/${NAME}.spec"
rpmbuild -bs ~/"rpmbuild/SPECS/${NAME}.spec"
echo '::endgroup::'

echo '::group::Testing RPM build'
rpmbuild --rebuild ~/"rpmbuild/SRPMS/${NAME}-*.src.rpm"
echo '::endgroup::'

echo '::group::Submitting to COPR'
copr-cli build log-courier2 ~/"rpmbuild/SRPMS/${NAME}-*.src.rpm"
echo '::endgroup::'
