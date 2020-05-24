#!/bin/bash

make collect 
make
go build -o metricbeat_0.0.8 main.go
GOOS=windows GOARCH=amd64 go build -o metricbeat_0.0.8.exe main.go
