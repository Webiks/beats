#!/bin/bash

make collect 
make
go build -o metricbeat_0.0.9 main.go
GOOS=windows GOARCH=amd64 go build -o metricbeat_0.0.9.exe main.go
