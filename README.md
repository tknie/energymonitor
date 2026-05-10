# EnergyMonitor application

- [EnergyMonitor application](#energymonitor-application)
  - [Introduction](#introduction)
  - [Environment variables](#environment-variables)
  - [Build](#build)
  - [Docker environment](#docker-environment)
  - [Usage in Grafana](#usage-in-grafana)

## Introduction

This application named energymonitor is used to query Ecoflow API getting current configuration and Solar data information and generate corresponding database tables in the Postgres database.

energymonitor queries periodically data from all devices quota API calls and store the data. The data is display in Grafana as an example.

Current Ecoflow devices are in use at the moment:

- Ecoflow Powerstream inverter
- Ecoflow Delta 2

In advance the application can listen for MQTT events. Unfortunately the MQTT is only active during usage of the Ecoflow App on mobile phone or smartphone device. The corresponding handling is not implemented yet.

## Environment variables

Variables | Default | Description
---------|----------|---------
 ECOFLOW_USER |  | Ecoflow API user
 ECOFLOW_PASSWORD |  | Ecoflow API password
 ECOFLOW_ACCESS_KEY |  | Ecoflow access key created in Ecoflow API
 ECOFLOW_SECRET_KEY |  | Ecoflow secret key created in Ecoflow API
 ECOFLOW_DB_URL |  | Postgres database URL used to store data
 ECOFLOW_DB_USER |  | Postgres user name
 ECOFLOW_DB_PASS |  | Postgres user password
 LOGPATH |  | Directory for log trace file
 energymonitor_WAIT_SECONDS | 30 | Time in seconds waiting between a loop reading statistic data in Ecoflow API

## Build

The `energymonitor` application is written in Golang. The tool can be build with

```sh
build.sh
```

## Docker environment

The energymonitor application and corresponding Postgres database is running in an Raspberry Pi.

Docker images are on Docker hub at

```docker
docker pull thknie/energymonitor:tagname
```

See the example script showing how to start the service with podman. Located is the script in this repostiory at [docker/podstart.sh](docker/podstart.sh).

## Usage in Grafana

In Grafana accessing the data source containing the data received by two input sources

- First sources comes from [mqtt2db](https://github.com/tknie/mqtt2db). This tool receives Tasmota electric meter data using an Mosquitto MQTT server and store it in database
- Second sources comes from [energymonitor](https://github.com/tknie/energymonitor) using the Ecoflow API receiving Solar panel and inverter statistics and store it in the same database

Both data are containing a wide range of statistic data which can be presented inside an Grafana Dashboard:

![Grafana Dashboard example](images/Grafana-power-statistics.png)

![Grafana Dashboard example](images/Grafana-solar-statistics.png)

______________________
These tools are provided as-is and without warranty or support. Users are free to use, fork and modify them, subject to the license agreement.
