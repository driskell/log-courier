#!/bin/bash
docker build -t driskell/log-courier:ppa registry-image
docker push driskell/log-courier:ppa
