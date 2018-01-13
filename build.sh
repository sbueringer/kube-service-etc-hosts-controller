#!/usr/bin/env bash

WORKDIR=`echo $0 | sed -e s/build.sh//`
cd ${WORKDIR}

TRAVIS_BUILD_DIR=${TRAVIS_BUILD_DIR:-"."}

export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

echo "Building linux binary"
env | grep GO
env | grep TRAVIS

FOLDER=/gopath/src/github.com/sbueringer/kube-service-etc-hosts-operator

rm -rf dist
mkdir dist

docker run -e GOOS=linux -e GOARCH=amd64 -e GOPATH=/gopath -e CGO_ENABLED=0 \
           -v $(pwd):$FOLDER \
           -v $(pwd)/dist:/dist \
           -w $FOLDER  \
           golang:1.9.2 \
           sh -c "go get -v -d ./...; \
                  go build -a -installsuffix cgo -gcflags '-N -l' -ldflags='-s -w' -v -o /dist/main ./cmd"

docker build -t docker.io/sbueringer/kube-service-etc-hosts-operator:latest .

if [ "$TRAVIS_BRANCH" == "master" ]
then
    docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD"
    docker push docker.io/sbueringer/kube-service-etc-hosts-operator:latest
fi