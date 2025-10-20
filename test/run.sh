#!/usr/bin/env sh

set -ex

# Source .envrc if provided or exists, otherwise use minio defaults
USE_MINIO=1
if [ -n "$1" ] && [ -f "$1" ]; then
  . "$1"
  USE_MINIO=0
else
  export AWS_ACCESS_KEY_ID="RnxRnuedwpQz4RjFUVeO"
  export AWS_REGION="us"
  export AWS_SECRET_ACCESS_KEY="zi5PIiyPwi0OWUwhIcWbGSLXCsLUwwv9SHTtl9fO"
  export S3_BUCKET="testbucket"
  export AWS_S3_ENDPOINT="http://127.0.0.1:9000"

  minio server /work/minio_data 2>&1 > /dev/null & sleep 2
  mcli alias set local http://localhost:9000 minioadmin minioadmin
  mcli admin user svcacct add --access-key "$AWS_ACCESS_KEY_ID" --secret-key "$AWS_SECRET_ACCESS_KEY" local minioadmin
  mcli mb "local/$S3_BUCKET"
fi

ROOT_PATH="${ROOT_PATH:-testroot}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Setup S3 client alias for verification
if [ "$USE_MINIO" -eq 1 ]; then
  # Configure mcli for real S3
  if [ -n "$AWS_S3_ENDPOINT" ]; then
    mcli alias set test "$AWS_S3_ENDPOINT" "$AWS_ACCESS_KEY_ID" "$AWS_SECRET_ACCESS_KEY"
  else
    mcli alias set test https://s3.amazonaws.com "$AWS_ACCESS_KEY_ID" "$AWS_SECRET_ACCESS_KEY"
  fi
  S3_ALIAS="test"
else
  S3_ALIAS="local"
fi

# Debug settings
DEBUG=1
export GIT_TRACE=$DEBUG
export GIT_TRANSFER_TRACE=$DEBUG
export GIT_CURL_VERBOSE=$DEBUG
export GIT_TRACE_PACKET=0

rm -rf /tmp/git-lfs-test
mkdir -p /tmp/git-lfs-test/fake-remote-repo
cd /tmp/git-lfs-test

git init --bare fake-remote-repo -b main
git clone --progress fake-remote-repo local-repo

# First repo: configure using commandline flags.
cd local-repo
git config core.autocrlf false
echo "# This is a lfs-s3 test." > README.md
git add README.md
git commit -m "Add pre-lfs commit."
git push origin main
git lfs install --local
git lfs track "*.bin"
git add .gitattributes
git commit -m "Adding .gitattributes"
git config --add lfs.customtransfer.lfs-s3.path "$SCRIPT_DIR/lfs-s3"
git config --add lfs.customtransfer.lfs-s3.args '--access_key_id='"$AWS_ACCESS_KEY_ID"' --secret_access_key='"$AWS_SECRET_ACCESS_KEY"' --bucket='"$S3_BUCKET"' --endpoint='"$AWS_S3_ENDPOINT"' --root_path='"$ROOT_PATH"
git config --add lfs.standalonetransferagent lfs-s3
git config --add lfs.concurrenttransfers 2

# Create test files
dd if=/dev/urandom of=blob1.bin bs=1024 count=1024
dd if=/dev/urandom of=blob2.bin bs=1024 count=1024
echo 'Simple, compressible text' > blob3.bin
git add blob*.bin
git commit -m "Adding files"
git push -q origin main

# Sanity check for the redundant upload avoidance feature
git lfs push --all origin main

git remote -v
cd ..

# Second repo: rely on AWS env variables.
GIT_LFS_SKIP_SMUDGE=1 git clone --progress fake-remote-repo local-repo-dup
cd local-repo-dup
git config core.autocrlf false
git lfs install --local
git config --add lfs.customtransfer.lfs-s3.path "$SCRIPT_DIR/lfs-s3"
git config --add lfs.customtransfer.lfs-s3.args --root_path="$ROOT_PATH"
git config --add lfs.standalonetransferagent lfs-s3
git config --add lfs.concurrenttransfers 2
git reset --hard main
git lfs pull
cd ..

# Verify files uploaded with correct compression
if [ "$USE_MINIO" -eq 1 ]; then
   FILE_COUNT=$(mcli ls "$S3_ALIAS/$S3_BUCKET/$ROOT_PATH/" | wc -l)
   if [ "$FILE_COUNT" -ne "3" ]; then
     echo "Unexpected number of files."
     exit 1
   fi

   ZSTD_COUNT=$(mcli ls "$S3_ALIAS/$S3_BUCKET/$ROOT_PATH/" | grep -c '\.zstd$' || true)
   if [ "$ZSTD_COUNT" -ne "3" ]; then
     echo "Unexpected number of zstd files."
     exit 1
   fi
fi

# Reupload with a different compression. ZSTD files should be removed, overwritten by gzip files.
cd local-repo-dup
git config --replace-all lfs.customtransfer.lfs-s3.args '--access_key_id='"$AWS_ACCESS_KEY_ID"' --secret_access_key='"$AWS_SECRET_ACCESS_KEY"' --bucket='"$S3_BUCKET"' --endpoint='"$AWS_S3_ENDPOINT"' --compression=gzip --root_path='"$ROOT_PATH"
git lfs push --all origin main
cd ..

if [ "$USE_MINIO" -eq 1 ]; then
   FILE_COUNT=$(mcli ls "$S3_ALIAS/$S3_BUCKET/$ROOT_PATH/" | wc -l)
   if [ "$FILE_COUNT" -ne "3" ]; then
     echo "Unexpected number of files."
     exit 1
   fi

   GZ_COUNT=$(mcli ls "$S3_ALIAS/$S3_BUCKET/$ROOT_PATH/" | grep -c '\.gz$' || true)
   if [ "$GZ_COUNT" -ne "3" ]; then
     echo "Unexpected number of gz files."
     exit 1
   fi
fi

# Ensure that we can re-download the LFS files even though their compression has changed.
cd local-repo
rm -rf .git/lfs/objects
git lfs fetch
cd ..

echo "PASS"
