#!/bin/bash
docker build --platform linux/amd64 -t driskell/log-courier:rpm registry-image
docker push driskell/log-courier:rpm
