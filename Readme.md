# Git LFS: S3 agent

[![Go Reference](https://pkg.go.dev/badge/github.com/nicolas-graves/lfs-s3.svg)](https://pkg.go.dev/github.com/nicolas-graves/lfs-s3)
![Build](https://github.com/nicolas-graves/lfs-s3/actions/workflows/build.yml/badge.svg)
![Test](https://github.com/nicolas-graves/lfs-s3/actions/workflows/test.yml/badge.svg)


## Installation

Run `go install github.com/nicolas-graves/lfs-s3@latest` or download the [corresponding release binary](https://github.com/nicolas-graves/lfs-s3/releases) to install the latest version on your system.

To clone a git repo that uses this (or any other non-default) LFS agent, use the following command: `GIT_LFS_SKIP_SMUDGE=1 git clone <source>`

Then [configure](#configuration) lfs-s3 and run `git lfs pull`.

To clone a git repo that uses this (or any other non-default) LFS agent, use the following command: `GIT_LFS_SKIP_SMUDGE=1 git clone <source>`

Then [configure](#configuration) izlfs-s3 and run `git lfs pull`.

## Details

`lfs-s3` is a [Custom Transfer
Agent](https://github.com/git-lfs/git-lfs/blob/master/docs/custom-transfers.md)
for [Git LFS](https://git-lfs.github.com/) which simply sends LFS
binary files to an [S3
bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html).

## Features

- Includes a workaround for GCS compatibility.
- Can be configured in two ways:
  - one-time (per repo) setup via `git config` without any environment variables needed later
  - AWS environment variables or config profile
- Compresses uploaded files.
- Avoids redundant re-uploads based on S3 checksumming.

## Configuration

> [!NOTE]
> For backwards-compatibility, AWS environment variables and/or config files can also be used for some configuration options. See the [Alternative configuration method](#alternative-configuration-method) section.

Command-line parameters can be used to configure every aspect of the tool, including S3 credentials.

> [!IMPORTANT]
> Your credentials will be exposed as cleartext in your git configuration file. I don't consider this a security issue, as this exposes your LFS/S3 data files only when your non-LFS git data files are exposed.

Run these commands in your git repo to configure your LFS S3 storage:
```sh
git config --add lfs.customtransfer.lfs-s3.path lfs-s3
git config --add lfs.standalonetransferagent lfs-s3
git config --add lfs.customtransfer.lfs-s3.args '--access_key_id=<S3 access key> --secret_access_key=<S3 secret key> --bucket=<S3 bucket> --endpoint=<S3 endpoint> --region=<optional S3 region>'
```

The full list of command-line flags:

| Name                      | Description                                                                                                           | Default value | Optional |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------- | ------------- | -------- |
| `--access_key_id`         | S3 Access key ID                                                                                                      |               | False    |
| `--secret_access_key`     | S3 Secret access key                                                                                                  |               | False    |
| `--bucket`                | S3 Bucket name                                                                                                        |               | False    |
| `--endpoint`              | S3 Endpoint                                                                                                           |               | False    |
| `--region`                | S3 Region                                                                                                             | `us`          | True     |
| `--root_path`             | Path within the bucket under which LFS files are uploaded. Can be empty.                                              |               | True     |
| `--delete_other_versions` | Whether to delete other (e.g. uploaded using different compression methods) versions of the stored file after upload. | `true`        | False    |
| `--compression`           | Compression to use for storing files. Possible values: zstd, gzip, none.                                              | `zstd`        | False    |

### Alternative configuration method

You should consider setting the following environment variables:
* `S3_BUCKET` - the bucket you wish to use for LFS storage. This
  variable is required in both cases.
* `AWS_REGION` - the region where your S3 bucket is.  If not provided
  the default from AWS SDK is used.
* `AWS_S3_ENDPOINT` - your S3 endpoint.  If not provided the default
  from AWS SDK is used.

Recommendations :
1) [Use shared credentials or config files](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html). In this case, you should also consider:
* `AWS_CONFIG_FILE` - in case you want to provide a project-specific config file.
* `AWS_PROFILE` - if one particular is set in the config file.

1) Use only environment variables. In this case, you have to also set:
* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`

If you set an environment variable, make sure you don't set the same command-line flag in the git lfs args configuration.

## Contribution

Pull requests are welcome.

## Testing

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
  out there, but they all seemed either outdated, unmaintained, or too
  complex (I don't need a running server to send a file!). It seems that
  this repo since influenced a few alternatives, namely:
  - [lfs-os](https://github.com/hacksadecimal/lfs-os)
  - [lfs-dal](https://github.com/regen100/lfs-dal). This one is
  recommended instead of this project if you're looking for a more
  capable solution.
  - Amazon's [git-remote-s3](https://github.com/awslabs/git-remote-s3)
