#!/bin/sh

go get -u ./...
go mod verify
go mod tidy
