#!/bin/bash

# Build the binary and then package it in a Docker image

TAG=$(date -u '+%Y%m%d%H%M')

# get dependencies
# compile for busybox (a self-contained binary)
# Do a Docker build
# Tag "latest" with what we just built

go get -d && \
    go build -a -tags netgo -installsuffix netgo -ldflags '-w' . && \
    docker build --tag dockchain/ingot:$TAG . && \
    docker tag -f dockchain/ingot:$TAG dockchain/ingot:latest 
