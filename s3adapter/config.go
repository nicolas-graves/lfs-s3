package s3adapter

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/infinitez-one/izlfs-s3/compression"
)

type Config struct {
	AccessKeyId         string
	SecretAccessKey     string
	Bucket              string
	Endpoint            string
	Region              string
	RootPath            string
	Compression         compression.Compression
	DeleteOtherVersions bool
}

func (config *Config) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{Source: "izlfs-s3",
		AccessKeyID:     config.AccessKeyId,
		SecretAccessKey: config.SecretAccessKey,
	}, nil
}
