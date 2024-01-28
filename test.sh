#!/usr/bin/env sh

# This file takes as an argument the .envrc file where variables are
# defined. It can be ommited if it is in the current directory.

# Test logs are in the linux /tmp/git-lfs-test directory.

if [ -z "$1" ]; then
  if [ -f .envrc ]; then
    ENVRC=".envrc"
  else
  echo "Please provide the path to the .envrc file"
  exit 1
  fi
else
  ENVRC="$1"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
[[ $ENVRC != /* ]] && ENVRC="$SCRIPT_DIR/$ENVRC"

# Use a wrapper to see input and output to the program.
go build &&\
echo -e "#!/usr/bin/env sh\n\ntee -a /tmp/git-lfs-test/input.log\
 | ${PWD}/lfs-s3 --debug 2> /tmp/git-lfs-test/error.log |\
 tee -a /tmp/git-lfs-test/output.log >&1\n" > lfs-s3.sh &&\
chmod +x lfs-s3.sh &&\
rm -rf /tmp/git-lfs-test &&\
mkdir -p /tmp/git-lfs-test/fake-remote-repo &&\
cd /tmp/git-lfs-test && (
  git init --bare fake-remote-repo &&\
  git clone --progress fake-remote-repo local-repo &&\
    cd local-repo && (
      echo "# This is a lfs-s3 test." > README.md
      git add README.md
      git commit -m "Add pre-lfs commit."
      git push origin master
      git lfs install --local
      git lfs track "*.bin"
      git add .gitattributes
      git commit -m "Adding .gitattributes"
      git config --add lfs.customtransfer.lfs-s3.path ../../lfs-s3.sh
      git config --add lfs.standalonetransferagent lfs-s3
      git config --add lfs.concurrenttransfers 2
      dd if=/dev/urandom of=blob1.bin bs=1024 count=1024
      dd if=/dev/urandom of=blob2.bin bs=1024 count=1024
      git add blob*.bin
      git commit -m "Adding files"
      source "$ENVRC"
      git push origin master
      git remote -v
    ) && cd ..
  git clone --progress fake-remote-repo local-repo-dup &&\
    cd local-repo-dup && (
      git lfs install --local
      git config --add lfs.customtransfer.lfs-s3.path ../../lfs-s3.sh
      git config --add lfs.standalonetransferagent lfs-s3
      git config --add lfs.concurrenttransfers 2
      source "$ENVRC"
      git reset --hard master
      git lfs pull
    )
)
