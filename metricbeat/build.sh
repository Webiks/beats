#!/bin/bash

make collect 
make
go build -o metricbeat_0.0.7 main.go
GOOS=windows GOARCH=amd64 go build -o metricbeat_0.0.7.exe main.go
