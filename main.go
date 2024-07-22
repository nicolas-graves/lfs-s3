package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/infinitez-one/izlfs-s3/compression"
	"github.com/infinitez-one/izlfs-s3/s3adapter"
	"github.com/infinitez-one/izlfs-s3/service"
)

var config s3adapter.Config
var comp string

func init() {
	flag.StringVar(&config.AccessKeyId, "access_key_id", "", "S3 Access Key ID")
	flag.StringVar(&config.SecretAccessKey, "secret_access_key", "", "S3 Secret Access Key")
	flag.StringVar(&config.Bucket, "bucket", "", "S3 Bucket")
	flag.StringVar(&config.Endpoint, "endpoint", "", "S3 Endpoint")
	flag.StringVar(&config.Region, "region", "us", "S3 Region")
	flag.StringVar(&config.RootPath, "root_path", "", "Path within the bucket under which LFS files are uploaded. Can be empty.")
	flag.BoolVar(&config.DeleteOtherVersions, "delete_other_versions", true, "Whether to delete other (e.g. uploaded using different compression methods) versions of the stored file after upload.")

	var compressions []string
	for _, c := range compression.Compressions {
		compressions = append(compressions, c.Name())
	}
	flag.StringVar(&comp, "compression", compression.Compressions[0].Name(), "Compression to use for storing files. Possible values: "+
		strings.Join(compressions, ", "))
}

func tryFromEnv(setting *string, key string) {
	if *setting == "" {
		*setting = os.Getenv(key)
	}
}

func run() error {
	for _, c := range compression.Compressions {
		if c.Name() == comp {
			config.Compression = c
			break
		}
	}

	// For backwards-compatibility, also allow using env variables.
	tryFromEnv(&config.Bucket, "S3_BUCKET")
	tryFromEnv(&config.Region, "AWS_REGION")
	tryFromEnv(&config.Endpoint, "AWS_S3_ENDPOINT")

	return service.Serve(os.Stdin, os.Stdout, os.Stderr, &config)
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
