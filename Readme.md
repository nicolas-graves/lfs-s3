# Git LFS: S3 agent

## What is it?

`lfs-s3` is a tiny (under 300 SLOC) [Custom Transfer
Agent](https://github.com/git-lfs/git-lfs/blob/master/docs/custom-transfers.md)
for [Git LFS](https://git-lfs.github.com/) which allows you to use a plain
folder as the remote storage location for all your large media files.

## Why?

Let's say you use Git, but you don't use any fancy hosting solution. You just
use a plain Git repo on a server somewhere, perhaps using SSH so you don't even
need a web server. It's simple and great.

But how do you use Git LFS? It usually wants a server to expose API endpoints.
Sure you could use one of the [big](https://bitbucket.org) [hosting](https://github.com)
[providers](https://gitlab.com), but that makes everything more complicated.

If you want a git repository that just works and simply sends its
binary files to an s3 provider, this is what this adapter does.
There are plenty of similar "solutions" out there, but they all seem
either outdated or too complex (I don't need to have a server running
just to send a file to S3).

If you already have plenty of storage sitting on a NAS somewhere, or via
Dropbox, Google Drive, you might instead want to check out
[lfs-folderstore](https://github.com/sinbad/lfs-folderstore).


## How to use

### Prerequisites

This has been tested with Git LFS 3.3.0. You probably need to be
running Git LFS version 2.3.0 or later.

### Download &amp; install

You will need `lfs-s3[.exe]` to be on your system path somewhere.

I haven't setup the releases yet, so you'll need to build it from
source using the standard `go build`. PR Welcome to help me setup
this.

### Environment variables

All S3 configuration options use environment variables. All of these
configuration variables must be set.

* `AWS_REGION` - the region where your S3 bucket is.
* `AWS_ACCESS_KEY_ID` - your access key.
* `AWS_SECRET_ACCESS_KEY` - your secret key.
* `AWS_S3_ENDPOINT` - your S3 endpoint.
* `S3_BUCKET` - the bucket you wish to use for LFS storage.
* `S3_USEPATHSTYLE` - boolean to set the S3 option [usePathStyle](https://docs.aws.amazon.com/AmazonS3/latest/userguide/dual-stack-endpoints.html#dual-stack-endpoints-description).

Although there is AWS in the environment variables, it should work
with any S3 provider, given it has the same configuration. I use OVH
for instance.

You can simply test if your S3 provider works with a `.envrc` file
given as an argument to the test file `test.sh`. Note that this will
upload a random 1mb binary to your bucket.

You can use what you want for this. I use [direnv](https://github.com/direnv/direnv).

### Configure a fresh repo

Starting a new repository is the easiest case.

* Initialise your repository as usual with `git init` and `git lfs track *.png` etc
* Create some commits with LFS binaries
* Add your plain git remote using `git remote add origin <url>`
* Run these commands to configure your LFS folder:
  * `git config --add lfs.customtransfer.lfs-s3.path lfs-s3`
  * `git config --add lfs.standalonetransferagent lfs-s3`
* `git push origin master` will now copy any media to that folder

A few things to note:

* The `standalonetransferagent` forced Git LFS to use the folder agent for all
  pushes and pulls. If you want to use another remote which uses the standard
  LFS API, you should see the next section.

### Configure an existing repo

(Warning) : This has been tested on simple repositories in `test.sh`,
but I can't get it to work on complex repositories. Comes with no
guarantee. Feedback welcome.

If you already have a Git LFS repository pushing to a standard LFS server, and
you want to either move to a folder, or replicate, it's a little more complicated.

* Create a new remote using `git remote add folderremote <url>`. Do this even if you want to keep the git repo at the same URL as currently.
* Run these commands to configure the folder store:
  * `git config --add lfs.customtransfer.lfs-s3.path lfs-s3`
  * `git config --add lfs.<url>.standalonetransferagent lfs-s3` - important: use the new Git repo URL
* `git push folderremote master ...` - important: list all branches you wish to keep LFS content for. Only LFS content which is reachable from the branches you list (at any version) will be copied to the remote

### Cloning a repo

There is one downside to this 'simple' approach to LFS storage - on cloning a
repository, git-lfs can't know how to fetch the LFS content, until you configure
things again using `git config`. That's the nature of the fact that you're using
a simple Git remote with no LFS API to expose this information.

It's not that hard to resolve though, you just need a couple of extra steps
when you clone fresh. Here's the sequence:

* `git clone <url> <folder>`
    * this will work for the git data, but will report "Error downloading object" when trying to get LFS data
* `cd <folder>` - to enter your newly cloned repo
* Configure as with a new repo:
  * `git config --add lfs.customtransfer.lfs-s3.path lfs-s3`
  * `git config --add lfs.standalonetransferagent lfs-folder`
* `git reset --hard master`
* `git lfs pull`

## Notes

* It's entirely up to you whether you use different S3 buckets per project, or
  share one between many projects. In the former case, it's easier to reclaim
  space by deleting a specific project, in the latter case you can save space if
  you have common files between projects (they'll have the same hash).
* This would not have been possible in Go (I had a python version)
  without the work done by Steve Streeting on
  [lfs-folderstore](https://github.com/sinbad/lfs-folderstore). Thanks
  to him! The license is therefore also MIT here.
* Upload and download progress report are implemented, but they only
  report for every 5 MB of data. This is currently hardcoded, as it's
  the limit value for my S3 provider. It can be put in an environment
  variable later if necessary.
* I don't use Windows. Please report issues if you experience them there.
