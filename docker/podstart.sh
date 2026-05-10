#!/bin/bash

#
# Copyright 2025-2026 Thorsten A. Knieling
#
# SPDX-License-Identifier: Apache-2.0
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#

POSTGRES_USER=<postgres user>
POSTGRES_PASSWORD=<postgres user password>
ECOFLOW_DB_URL=<postgres db URL>
ECOFLOW_USER=<email>
ECOFLOW_PASSWORD=<email API password>
ECOFLOW_DEVICE_SN=<Device SN>
ECOFLOW_ACCESS_KEY=<access key>
ECOFLOW_SECRET_KEY=<secret key>
DOCKER_IMAGE=<docker image>
PWD=`pwd`

podman pod create \
   --name energymonitor_pod
podman run --name energymonitor --pod energymonitor_pod \
        -e TZ="Europe/Berlin" \
        --restart=on-failure:10 \
        -e ECOFLOW_USER=${ECOFLOW_USER} \
        -e ECOFLOW_PASSWORD=${ECOFLOW_PASSWORD} \
        -e ECOFLOW_DEVICE_SN=${ECOFLOW_DEVICE_SN} \
        -e ECOFLOW_ACCESS_KEY=${ECOFLOW_ACCESS_KEY} \
        -e ECOFLOW_SECRET_KEY=${ECOFLOW_SECRET_KEY} \
        -e ECOFLOW_DB_URL=${ECOFLOW_DB_URL} \
        -e ECOFLOW_DB_USER=${POSTGRES_USER} \
        -e ECOFLOW_DB_PASS=${POSTGRES_PASSWORD} \
        -e LOGPATH=/energymonitor/log \
        -v $PWD/energymonitor/log:/energymonitor/log:rw \
        -d ${DOCKER_IMAGE}
