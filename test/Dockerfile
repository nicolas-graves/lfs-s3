FROM alpine:3.20.1

RUN apk add --no-cache curl git git-lfs minio minio-client

RUN git config --global user.email "izlfs-s3-test@none.com"
RUN git config --global user.name "izlfs-s3 test"
RUN git config --global init.defaultBranch main

RUN mkdir /work
WORKDIR /work
RUN mkdir -p /work/minio_data
RUN chown -R minio:minio /work/minio_data
COPY izlfs-s3 /work/izlfs-s3
COPY run.sh /work/run.sh
CMD /work/run.sh
