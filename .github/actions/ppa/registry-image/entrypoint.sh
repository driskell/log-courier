#!/bin/bash

set -eo pipefail

VERSION=${VERSION#refs/tags/}
RELEASE=1
DRELEASE=${DRELEASE}

git config --global safe.directory /github/workspace

export PATH="/usr/local/go/bin:$PATH"

echo "::group::Checking $NAME exists in $REF"
if [ "${NAME}" != "log-courier" ] && [ ! -d "${NAME}" ]; then
	exit 0
fi
echo '::endgroup::'

if [ "${DRELEASE}" != 1 ]; then
	echo "::group::Downloading original source for ${VERSION#v}-${RELEASE} from previous release"
	wget --progress dot:giga "https://launchpad.net/~devel-k/+archive/ubuntu/log-courier2/+sourcefiles/${NAME}/${VERSION#v}-${RELEASE}~trusty$(( DRELEASE - 1 ))/${NAME}_${VERSION#v}.orig.tar.gz" -O ~/"${NAME}_${VERSION#v}.orig.tar.gz"
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
	LC_DEFAULT_CONFIGURATION_FILE=/etc/log-courier/log-courier.yaml \
		LC_DEFAULT_GENERAL_PERSIST_DIR=/var/lib/log-courier \
		LC_DEFAULT_ADMIN_BIND=unix:/var/run/log-courier/admin.socket \
		go generate -mod=vendor .

	LC_DEFAULT_CONFIGURATION_FILE=/etc/log-carver/log-carver.yaml \
		LC_DEFAULT_GENERAL_PERSIST_DIR=/var/lib/log-carver \
		LC_DEFAULT_ADMIN_BIND=unix:/var/run/log-carver/admin.socket \
		go generate -mod=vendor ./log-carver

	LC_DEFAULT_CONFIGURATION_FILE=/etc/log-courier/log-courier.yaml \
		LC_DEFAULT_CARVER_CONFIGURATION_FILE=/etc/log-carver/log-courier.yaml \
		LC_DEFAULT_CARVER_ADMIN_BIND=unix:/var/run/log-carver/admin.socket \
		go generate -mod=vendor ./lc-admin

	mkdir "$(pwd)/bin-i386" "$(pwd)/bin-amd64" "$(pwd)/bin-arm64"
	GOARCH=386 go build -mod=vendor -o "$(pwd)/bin-i386" . ./lc-admin ./log-carver ./lc-tlscert
	GOARCH=amd64 go build -mod=vendor -o "$(pwd)/bin-amd64" . ./lc-admin ./log-carver ./lc-tlscert
	GOARCH=arm64 go build -mod=vendor -o "$(pwd)/bin-arm64" . ./lc-admin ./log-carver ./lc-tlscert
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
D6AB748910B1CF77C9A0A5BD3B83B8DE48FE9B1E:6:
EOF
echo '::endgroup::'

for DIST in trusty xenial bionic focal jammy noble oracular; do
	echo "::group::Preparing debian package for $DIST"
	rm -rf ~/"${NAME}"
	cd ~
	tar -xzf ~/"${NAME}_${VERSION#v}.orig.tar.gz"
	cd ~/"${NAME}"
	rm -rf debian
	if [ -d "/github/workspace/.main/contrib/ppa/${NAME}" ]; then
		cp -rf "/github/workspace/.main/contrib/ppa/${NAME}" debian
	elif [ "$DIST" == "trusty" ]; then
		cp -rf "/github/workspace/.main/contrib/ppa/${NAME}-upstart" debian
	elif [ "$DIST" == "xenial" ] || [ "$DIST" == "bionic" ] || [ "$DIST" == "focal" ]; then
		cp -rf "/github/workspace/.main/contrib/ppa/${NAME}-systemd" debian
	else
		cp -rf "/github/workspace/.main/contrib/ppa/${NAME}-systemd-v13" debian
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
	mkdir -p "$GITHUB_WORKSPACE"/artifacts
	cp -rf "../${NAME}_${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}"* "$GITHUB_WORKSPACE"/artifacts/
	echo '::endgroup::'

	echo "::group::Testing DEB"
	debuild -d -b
	echo '::endgroup::'

	if [ -n "${SKIP_SUBMIT}" ]; then
		continue
	fi

	echo "::group::Submitting package"
	dput ppa:devel-k/log-courier2 "../${NAME}_${VERSION#v}-${RELEASE}~${DIST}${DRELEASE}_source.changes"
	echo '::endgroup::'
done
