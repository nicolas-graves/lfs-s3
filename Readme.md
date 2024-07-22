# Git LFS: S3 agent

## Origin

Originally forked from https://github.com/nicolas-graves/lfs-s3, this has been rewritten quite significantly.

## Details

`izlfs-s3` is a [Custom Transfer
Agent](https://github.com/git-lfs/git-lfs/blob/master/docs/custom-transfers.md)
for [Git LFS](https://git-lfs.github.com/) which simply sends LFS
binary files to an [S3
bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html).

> [!IMPORTANT]
> This has been developed and tested primaraily for Google Cloud Storage.

## Features

- Includes a workaround for GCS compatibility.
- Can be configured in two ways:
  - one-time (per repo) setup via `git config` without any environment variables needed later
  - AWS environment variables or config profile
- Compresses uploaded files.
- Avoids redundant re-uploads based on S3 checksumming.

## Download & install

You need `izlfs-s3[.exe]` to be on your system path somewhere.

I haven't setup the releases yet, so you'll need to build it from
source using the standard `go build`.

## Configuration

> [!NOTE]
> For backwards-compatibility, AWS environment variables and/or config files can also be used for some configuration options. See the [Alternative configuration method](#alternative-configuration-method) section.

Command-line parameters can be used to configure every aspect of the tool, including S3 credentials.

> [!IMPORTANT]
> Your credentials will be exposed as cleartext in your git configuration file. I don't consider this a security issue, as this exposes your LFS/S3 data files only when your non-LFS git data files are exposed.

Run these commands in your git repo to configure your LFS S3 storage:
```sh
git config --add lfs.customtransfer.izlfs-s3.path izlfs-s3
git config --add lfs.standalonetransferagent izlfs-s3
git config --add lfs.customtransfer.izlfs-s3.args '--access_key_id=<S3 access key> --secret_access_key=<S3 secret key> --bucket=<S3 bucket> --endpoint=<S3 endpoint> --region=<optional S3 region>'
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
