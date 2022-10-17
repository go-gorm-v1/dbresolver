#!/bin/bash

if [[ -z "$1" ]]; then
    go test -v ./...
elif [[ "$1" == "all" ]]; then
    go test -v ./...
else
    go test -v "github.com/go-gorm-v1/$1"
fi

