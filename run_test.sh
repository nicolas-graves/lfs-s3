#!/bin/sh
set -ex

go build -o test/lfs-s3
cd test
docker build . -t lfs-s3-test
docker run --rm -it lfs-s3-test
