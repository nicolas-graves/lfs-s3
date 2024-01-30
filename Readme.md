# Git LFS: S3 agent

Let's say you use a simple Git repo, without any fancy hosting
solution. How do you use Git LFS? Sure you could use one of the
[big](https://bitbucket.org) [hosting](https://github.com)
[providers](https://gitlab.com), but that makes everything more
complicated.

`lfs-s3` is a tiny (~300 SLOC) [Custom Transfer
Agent](https://github.com/git-lfs/git-lfs/blob/master/docs/custom-transfers.md)
for [Git LFS](https://git-lfs.github.com/) which simply sends LFS
binary files to an [S3
bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html).

## How to use

### Download &amp; install

You need `lfs-s3[.exe]` to be on your system path somewhere.

I haven't setup the releases yet, so you'll need to build it from
source using the standard `go build`. PR Welcome to help me setup
this.

### Configuration

The
[default](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials)
AWS SDK mechanisms are used for gathering credentials (though IAM
roles are untested yet). Your S3 provider doesn't have to be AWS, most
providers implement the same API.

You should consider setting the following environment variables:
* `S3_BUCKET` - the bucket you wish to use for LFS storage. This
  variable is required in both cases.
* `AWS_REGION` - the region where your S3 bucket is.  If not provided
  the default from AWS SDK is used.
* `AWS_S3_ENDPOINT` - your S3 endpoint.  If not provided the default
  from AWS SDK is used.
* `S3_USEPATHSTYLE` - boolean to set the S3 option [usePathStyle](https://docs.aws.amazon.com/AmazonS3/latest/userguide/dual-stack-endpoints.html#dual-stack-endpoints-description).
* `S3_BUCKET_CDN` - in case you use a [CDN](https://aws.amazon.com/what-is/cdn) to improve download speed.

Recommendations :
1) [Use shared credentials or config files](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html). In this case, you should also consider:
* `AWS_CONFIG_FILE` - in case you want to provide a project-specific config file.
* `AWS_PROFILE` - if one particular is set in the config file.

2) Use only environment variables. In this case, you have to also set:
* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`

Some tools are available to manage environment variables seamlessly,
[direnv](https://github.com/direnv/direnv) is recommended.

### Testing

You can simply test if your S3 provider works with a `.envrc` file
given as an argument to the test file `test.sh`. Note that this will
upload a random 1mb binary to your bucket.

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

* This has been tested with 3.4.0 >= Git LFS >= 3.3.0. Earlier version
  will not be supported.
* It's entirely up to you whether you use different S3 buckets per project, or
  share one between many projects. In the former case, it's easier to reclaim
  space by deleting a specific project, in the latter case you can save space if
  you have common files between projects (they'll have the same hash).
* This work benefited a lot from
  [lfs-folderstore](https://github.com/sinbad/lfs-folderstore),
  thanks!
* Upload and download progress report are implemented, but they only
  report for every 5 MB of data. This is currently hardcoded, as it's
  the limit value for my S3 provider. It can be put in an environment
  variable later if necessary.
* Relation to other tools : There were plenty of similar "solutions"
  out there, but they all seem either outdated or too complex (I don't
  need a running server to send a file!). Seems that this work
  influenced a few alternatives, namely
  https://github.com/hacksadecimal/lfs-os then
  https://github.com/regen100/lfs-dal . `lfs-dal` is very promising
  and will probably be recommended instead of this project in a near
  future.
