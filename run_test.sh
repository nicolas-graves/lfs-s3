#!/bin/sh
set -ex

CGO_ENABLED=0 GOOS=linux go build -o test/lfs-s3
cd test
docker build . -t lfs-s3-test
docker run --rm -t lfs-s3-test
