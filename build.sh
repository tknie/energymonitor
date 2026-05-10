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

VERSION=${VERSION:-v1.0.0}
PACKAGE=github.com/tknie/energymonitor
DATE=$(date +%d-%m-%Y'_'%H:%M:%S)
GO_FLAGS=

mkdir -p bin/plugins

for i in plugins/*; do
	if [ -d "$i" ]; then
		plugin_name=$(basename "$i")
		go build ${GO_FLAGS} \
			-buildmode=plugin \
			-ldflags '-X $(PACKAGE).Version=$(VERSION) -X $(PACKAGE).BuildDate=$(DATE) -s -w' \
			-o "bin/plugins/${plugin_name}.so" "./plugins/${plugin_name}"
	fi
done
#go build ${GO_FLAGS} \
#	    -buildmode=plugin \
#	    -ldflags '-X $(PACKAGE).Version=$(VERSION) -X $(PACKAGE).BuildDate=$(DATE) -s -w' \
#	    -o bin/plugins/ecoflow.so ./plugins/ecoflow
go build -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.BuildDate=${DATE}" -o bin/energymonitor ./cmd/energymonitor
