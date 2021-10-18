#!/bin/bash
docker build -t driskell/log-courier:rpm registry-image
docker push driskell/log-courier:rpm
