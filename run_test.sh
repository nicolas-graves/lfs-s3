#!/bin/sh
set -ex

go build -o test/izlfs-s3
cd test
docker build . -t izlfs-s3-test
docker run --rm -it izlfs-s3-test
