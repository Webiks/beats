#!/bin/bash

make collect 
make
go build -o metricbeat_0.0.10 main.go
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc  go build -o metricbeat_0.0.10.exe main.go
