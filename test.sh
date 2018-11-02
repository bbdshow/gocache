#!/bin/bash
set -e
for d in $(go list ./... | grep -E 'gocache$'); do
    go test -v -coverprofile=coverage.txt -covermode=count
done

for d in $(go list ./... | grep -E 'gocache$'); do
   go test -race -covermode=atomic -coverprofile=coverage.txt 
done