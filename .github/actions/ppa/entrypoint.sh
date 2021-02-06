#!/bin/bash

set -eo pipefail

RELEASE=1
DRELEASE=1

echo "::group::Checking exists in $VERSION"
if [ "${NAME}" != "log-courier" ] && [ ! -d "/github/workspace/${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

echo "::group::Generating sources for $VERSION"
GIT_DIR=/github/workspace/.git git archive --format=tar.gz --output ~/"${NAME}_${VERSION#v}.orig.tar.gz" --prefix "${NAME}/" "$VERSION"
tar -C ~ -xzf ~/"${NAME}_${VERSION#v}.orig.tar.gz"
echo '::endgroup::'

echo '::group::Installing secrets'
base64 -id <<<"$GNU_PG" | gpg --batch --import
echo '::endgroup::'

cd ~/"${NAME}"

for DIST in trusty xenial bionic eoan focal; do
	echo "::group::Preparing debian package for $DIST"
	rm -rf debian
	if [ "$DIST" == "trusty" ]; then
		if [ -d contrib/ppa/ ]; then
			cp -rf contrib/ppa/deb-upstart debian
		else
			cp -rf contrib/deb-upstart debian
		fi
	else
		if [ -d contrib/ppa/ ]; then
			cp -rf contrib/ppa/deb-systemd debian
		else
			cp -rf contrib/deb-systemd debian
		fi
	fi
	debchange \
		--newversion "${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}" \
		--allow-lower-version "${VERSION#v}-${RELEASE}~*" \
		--distribution "$DIST" \
		--controlmaint \
		"Package for ${DIST}"
	echo '::endgroup::'

	echo "::group::Building package"
	debuild -d -S -sa
	echo '::endgroup::'

	echo "::group::Submitting package"
	dput ppa:devel-k/log-courier2 "../${NAME}_${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}_source.changes"
	echo '::endgroup::'
done
