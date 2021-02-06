#!/bin/bash

set -eo pipefail

VERSION=${VERSION#refs/tags/}
RELEASE=1
DRELEASE=1

echo "::group::Checking exists in $VERSION"
if [ "${NAME}" != "log-courier" ] && [ ! -d "${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

echo "::group::Generating sources for $VERSION"
git archive --format=tar.gz --output ~/"${NAME}_${VERSION#v}.orig.tar.gz" --prefix "${NAME}/" "$VERSION"
go mod vendor
tar -rzf ~/"${NAME}_${VERSION#v}.orig.tar.gz" vendor
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
		cp -rf ".master/contrib/ppa/${NAME}-upstart" debian
	else
		cp -rf ".master/contrib/ppa/${NAME}-systemd" debian
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
