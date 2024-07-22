#!/bin/sh
set -ex

CGO_ENABLED=0 GOOS=linux go build -o test/izlfs-s3
cd test
docker build . -t izlfs-s3-test
docker run --rm -t izlfs-s3-test
