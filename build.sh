#!/bin/bash
GOOS=windows GOARCH=amd64 go build -o winmessage-test.exe main.go
