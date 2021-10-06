#!/bin/bash

# Build the docker image
docker build -t $1 . -f ./Dockerfile
