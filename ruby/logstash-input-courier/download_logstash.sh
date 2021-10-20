#!/bin/bash

set -eo pipefail

SOURCE="https://artifacts.elastic.co/downloads/logstash/logstash-7.7.0.tar.gz"
CHECKSUM="970740adc47551d7967b9841cc39d15f2cbdcd46c2fee1f84b5688fac266fdcd2202cbb10d3a10cf3768606f693ed2e4fc79e91d293a3295083718bafaa7bc9d"

echo "===== Downloading ====="
wget "$SOURCE" -O logstash.tar.gz
echo "===== Verifying checksum ====="
shasum -a 512 -c <(echo "$CHECKSUM  logstash.tar.gz")
echo "===== Extracting ====="
mkdir -p logstash
tar -C logstash --strip-components 1 -xzf logstash.tar.gz
echo "===== Testing ====="
./logstash/bin/logstash --version
./logstash/vendor/jruby/bin/jruby --version
echo "===== Completed ====="
