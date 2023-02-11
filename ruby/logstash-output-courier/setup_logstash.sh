#!/bin/bash

if [ -d logstash ]; then
	rm -rf logstash
fi

git clone -b v8.6.1 https://github.com/elastic/logstash.git logstash --depth 1
