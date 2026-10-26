#!/bin/bash

set -eo pipefail

if [ -d logstash ]; then
	rm -rf logstash
fi

git clone -b v8.19.23 https://github.com/elastic/logstash.git logstash --depth 1

(
	cd logstash
	./gradlew installDevelopmentGems
)
