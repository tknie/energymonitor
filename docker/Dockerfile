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

FROM alpine:latest

RUN  apk update && apk upgrade && apk add gcompat

COPY energymonitor /energymonitor/bin/
COPY docker-entrypoint.sh /

RUN chmod a+x docker-entrypoint.sh && \
   mkdir /energymonitor/log && chmod 777 /energymonitor/log

ENV LOGFILE=/energymonitor/log/

CMD ["/docker-entrypoint.sh"]
