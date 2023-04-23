package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"

	"git.sr.ht/~ngraves/lfs-s3/api"
	"git.sr.ht/~ngraves/lfs-s3/util"
)

type Message struct {
	Event  string  `json:"event"`
	Oid    string  `json:"oid"`
	Size   *int64  `json:"size,omitempty"`
	Path   string  `json:"path,omitempty"`
	Action string  `json:"action,omitempty"`
	Error  *Error  `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Serve(stdin io.Reader, stdout, stderr io.Writer) {
	scanner := bufio.NewScanner(stdin)
	writer := bufio.NewWriter(stdout)
	errWriter := bufio.NewWriter(stderr)

	for scanner.Scan() {
		line := scanner.Text()
		var req api.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error reading input: %s\n", err))
			return
		}

		switch req.Event {
		case "init":
			resp := &api.InitResponse{}
			api.SendResponse(resp, writer, errWriter)
		case "download":
			util.WriteToStderr(fmt.Sprintf("Received download request for %s\n", req.Oid), errWriter)
			retrieve(req.Oid, req.Size, req.Action, writer, errWriter)
		case "upload":
			util.WriteToStderr(fmt.Sprintf("Received upload request for %s\n", req.Oid), errWriter)
			store(req.Oid, req.Size, req.Action, writer, errWriter)
		case "terminate":
			util.WriteToStderr("Terminating test custom adapter gracefully.\n", errWriter)
			break
		}
	}
}

func createS3Client() *s3.Client {
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	endpointURL := os.Getenv("AWS_S3_ENDPOINT")

	cfg, _ := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, _ string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpointURL, SigningRegion: region}, nil
			})),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
	)

	return s3.NewFromConfig(cfg)
}

func retrieve(oid string, size int64, action *api.Action, writer, errWriter *bufio.Writer) {
	client := createS3Client()
	bucketName := os.Getenv("S3_BUCKET")

	localPath := fmt.Sprintf("%s.tmp", oid)
	file, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer file.Close()

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 10 * 1024 * 1024     // 10 MB part size
		d.Concurrency = 3                 // Concurrent downloads
	})

	_, err = downloader.Download(context.Background(), file, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
	})

	if err != nil {
		return
	}

	complete := &api.TransferResponse{Event: "complete", Oid: oid, Path: localPath, Error: nil}
	err = api.SendResponse(complete, writer, errWriter)
	if err != nil {
		util.WriteToStderr(fmt.Sprintf("Unable to send completion message: %v\n", err), errWriter)
	}
}

func store(oid string, size int64, action *api.Action, writer, errWriter *bufio.Writer) {
	client := createS3Client()
	bucketName := os.Getenv("S3_BUCKET")

	localPath := ".git/lfs/objects/" + oid[:2] + "/" + oid[2:4] + "/" + oid
	file, err := os.Open(localPath)
	if err != nil {
		util.WriteToStderr(fmt.Sprintf("Error opening file: %v\n", err), errWriter)
		return
	}
	defer file.Close()

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024     // 10 MB part size
		u.LeavePartsOnError = true        // Keep uploaded parts on error
		u.Concurrency = 3                 // Concurrent uploads
	})

	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
		Body:   file,
	})

	if err != nil {
		util.WriteToStderr(fmt.Sprintf("Error uploading file: %v\n", err), errWriter)
		return
	}

	complete := &api.TransferResponse{Event: "complete", Oid: oid, Error: nil}
	err = api.SendResponse(complete, writer, errWriter)
	if err != nil {
		util.WriteToStderr(fmt.Sprintf("Unable to send completion message: %v\n", err), errWriter)
	}
}
