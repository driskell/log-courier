#!/bin/bash

set -eo pipefail

VERSION=${VERSION#refs/tags/}
RELEASE=1
DRELEASE=${DRELEASE}

echo "::group::Checking exists in $REF"
if [ "${NAME}" != "log-courier" ] && [ ! -d "${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

if [ "${DRELEASE}" != 1 ]; then
	echo "::group::Downloading original source for ${VERSION#v}-${RELEASE} from previous release"
	wget -q "https://launchpad.net/~devel-k/+archive/ubuntu/log-courier2/+sourcefiles/log-courier/${VERSION#v}-${RELEASE}~trusty$(( DRELEASE - 1 ))/${NAME}_${VERSION#v}.orig.tar.gz" -O ~/"${NAME}_${VERSION#v}.orig.tar.gz"
	echo '::endgroup::'
else
	echo "::group::Generating sources for $REF"
	git archive --format=tar.gz --output ~/"${NAME}_${VERSION#v}.orig.tar.gz" --prefix "${NAME}/" "$REF"
	tar -C ~ -xzf ~/"${NAME}_${VERSION#v}.orig.tar.gz"
	echo '::endgroup::'
	echo '::group::Adding vendored modules'
	cd ~/"${NAME}"
	go mod vendor
	cd -
	echo '::endgroup::'
	echo '::group::Precompiling via cross-compilation as no Go version exists in PPA that is suitable for all distributions'
	cd ~/"${NAME}"
	# Configure platform specific defaults
	export LC_DEFAULT_CONFIGURATION_FILE=/etc/log-courier/log-courier.yaml
	export LC_DEFAULT_GENERAL_PERSIST_DIR=/var/lib/log-courier
	export LC_DEFAULT_ADMIN_BIND=unix:/var/run/log-courier/admin.socket
	go generate -mod=vendor . ./lc-admin ./log-carver
	mkdir "$(pwd)/bin-i386" "$(pwd)/bin-amd64"
	GOARCH=386 go build -mod=vendor -o "$(pwd)/bin-i386" . ./lc-admin ./log-carver ./lc-tlscert
	GOARCH=amd64 go build -mod=vendor -o "$(pwd)/bin-amd64" . ./lc-admin ./log-carver ./lc-tlscert
	cd -
	echo '::endgroup::'
	echo '::group::Updating sources'
	# Clear cache after vendoring, so that the subsequent test DEB build does not try to use a VCS cache
	# This will allow us to detect vendoring issues as we will then see it attempt to download additional items
	go clean -cache -modcache -i -r
	tar -C ~ -czf ~/"${NAME}_${VERSION#v}.orig.tar.gz" "${NAME}"
	echo '::endgroup::'
fi

echo '::group::Installing secrets'
base64 -id <<<"$GNU_PG" | gpg --batch --import
gpg --import-ownertrust <<EOF
CF713BECBB9DA51E892E8AD0117FF0FC7420BA3F:6:
EOF
echo '::endgroup::'

for DIST in trusty xenial bionic focal; do
	echo "::group::Preparing debian package for $DIST"
	rm -rf ~/"${NAME}"
	cd ~
	tar -xzf ~/"${NAME}_${VERSION#v}.orig.tar.gz"
	cd ~/"${NAME}"
	rm -rf debian
	if [ "$DIST" == "trusty" ]; then
		cp -rf "/github/workspace/.master/contrib/ppa/${NAME}-upstart" debian
	else
		cp -rf "/github/workspace/.master/contrib/ppa/${NAME}-systemd" debian
	fi
	debchange \
		--create \
		--package "${NAME}" \
		--newversion "${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}" \
		--distribution "${DIST}" \
		--controlmaint \
		"Package for ${DIST}"
	echo '::endgroup::'

	echo "::group::Building source"
	debuild -d -S -sa
	echo '::endgroup::'

	echo "::group::Testing DEB"
	debuild -d -b
	echo '::endgroup::'

	echo "::group::Submitting package"
	dput ppa:devel-k/log-courier2 "../${NAME}_${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}_source.changes"
	echo '::endgroup::'
done
