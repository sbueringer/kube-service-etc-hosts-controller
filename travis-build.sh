#!/usr/bin/env bash

TRAVIS_BUILD_DIR=${TRAVIS_BUILD_DIR:-dist}

export GOOS=linux
export GOARCH=amd64

echo "Building linux binary"
env | grep GO
go build -ldflags='-s -w' -v -o $TRAVIS_BUILD_DIR/main ./cmd

docker build -t sbueringer/kube-service-etc-hosts-operator .
