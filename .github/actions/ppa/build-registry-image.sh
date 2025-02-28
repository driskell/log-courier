#!/bin/bash
docker build --platform linux/amd64 -t driskell/log-courier:ppa registry-image
docker push driskell/log-courier:ppa
