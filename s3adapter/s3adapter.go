package s3adapter

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const partSize = 5 * 1024 * 1024 // Size of transferred parts, in bytes.

type Connection struct {
	client *s3.Client
	config *Config
}

func (conn *Connection) asLfsPath(path string) string {
	root := conn.config.RootPath
	if root == "" {
		return path
	} else {
		return root + "/" + path
	}
}

func (conn *Connection) fileExists(path string) bool {
	ho, err := conn.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(conn.config.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return false
	}
	if ho == nil {
		return false
	}
	if ho.ContentLength == nil {
		return false
	}
	return true
}

func createS3Client(conf *Config) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithSharedConfigProfile(os.Getenv("AWS_PROFILE")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}
	cfg.BaseEndpoint = aws.String(conf.Endpoint)
	cfg.Region = conf.Region

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if conf.AccessKeyId != "" {
			o.Credentials = conf
		}
		if strings.Contains(conf.Endpoint, "storage.googleapis.com") {
			ignoreSigningHeaders(o, []string{"Accept-Encoding"})
		}
	}), nil
}

func New(config *Config) (*Connection, error) {
	c, err := createS3Client(config)
	if err != nil {
		return nil, err
	}
	ret := &Connection{
		client: c,
		config: config,
	}
	return ret, nil
}
