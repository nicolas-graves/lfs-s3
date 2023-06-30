package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"git.sr.ht/~ngraves/lfs-s3/api"
	"github.com/joho/godotenv"
)

type writerAtWrapper struct {
	w io.Writer
}

func (waw *writerAtWrapper) WriteAt(p []byte, off int64) (n int, err error) {
	return waw.w.Write(p)
}

type progressTracker struct {
	Reader         io.Reader
	Writer         io.WriterAt
	Oid            string
	TotalSize      int64
	RespWriter     io.Writer
	ErrWriter      io.Writer
	bytesProcessed int64
}

func (rw *progressTracker) Read(p []byte) (n int, err error) {
	n, err = rw.Reader.Read(p)
	if n > 0 {
		rw.bytesProcessed += int64(n)
		api.SendProgress(rw.Oid, rw.bytesProcessed, n, rw.RespWriter, rw.ErrWriter)
	}
	return
}

func (rw *progressTracker) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = rw.Writer.WriteAt(p, off)
	if n > 0 {
		rw.bytesProcessed += int64(n)
		api.SendProgress(rw.Oid, rw.bytesProcessed, n, rw.RespWriter, rw.ErrWriter)
	}
	return
}

func Serve(stdin io.Reader, stdout, stderr io.Writer) {
	scanner := bufio.NewScanner(stdin)
	writer := io.Writer(stdout)

	for scanner.Scan() {
		line := scanner.Text()
		var req api.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(stderr, fmt.Sprintf("Error reading input: %s\n", err))
			return
		}

		switch req.Event {
		case "init":
			resp := &api.InitResponse{}
			api.SendResponse(resp, writer, stderr)
		case "download":
			fmt.Fprintf(stderr, fmt.Sprintf("Received download request for %s\n", req.Oid))
			retrieve(req.Oid, req.Size, writer, stderr)
		case "upload":
			fmt.Fprintf(stderr, fmt.Sprintf("Received upload request for %s\n", req.Oid))
			store(req.Oid, req.Size, writer, stderr)
		case "terminate":
			fmt.Fprintf(stderr, "Terminating test custom adapter gracefully.\n")
			break
		}
	}
}

func createS3Client() *s3.Client {
	godotenv.Load(".env")
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

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		usePathStyle, err := strconv.ParseBool(os.Getenv("S3_USEPATHSTYLE"))
		if err != nil {
			usePathStyle = false
		}
		o.UsePathStyle = usePathStyle
	})
}

func retrieve(oid string, size int64, writer io.Writer, stderr io.Writer) {
	client := createS3Client()
	bucketName := os.Getenv("S3_BUCKET")

	localPath := ".git/lfs/objects/" + oid[:2] + "/" + oid[2:4] + "/" + oid
	file, err := os.Create(localPath)
	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Error creating file: %v\n", err))
		return
	}
	defer func() {
		file.Sync()
		file.Close()
	}()

	waw := &writerAtWrapper{file}
	progressWriter := &progressTracker{
		Writer:     waw,
		Oid:        oid,
		TotalSize:  size,
		RespWriter: writer,
		ErrWriter:  stderr,
	}

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 5 * 1024 * 1024 // 1 MB part size
		d.Concurrency = 1            // Concurrent downloads
	})

	_, err = downloader.Download(context.Background(), progressWriter, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
	})

	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Error downloading file: %v\n", err))
		return
	}

	complete := &api.TransferResponse{Event: "complete", Oid: oid, Path: localPath, Error: nil}
	err = api.SendResponse(complete, writer, stderr)
	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Unable to send completion message: %v\n", err))
	}
}

func store(oid string, size int64, writer io.Writer, stderr io.Writer) {
	client := createS3Client()
	bucketName := os.Getenv("S3_BUCKET")

	localPath := ".git/lfs/objects/" + oid[:2] + "/" + oid[2:4] + "/" + oid
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Error opening file: %v\n", err))
		return
	}
	defer func() {
		file.Sync()
		file.Close()
	}()

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024 // 1 MB part size
		// u.LeavePartsOnError = true        // Keep uploaded parts on error
	})

	progressReader := &progressTracker{
		Reader:     file,
		Oid:        oid,
		TotalSize:  size,
		RespWriter: writer,
		ErrWriter:  stderr,
	}

	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
		Body:   progressReader,
	})

	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Error uploading file: %v\n", err))
		return
	}

	complete := &api.TransferResponse{Event: "complete", Oid: oid, Error: nil}
	err = api.SendResponse(complete, writer, stderr)
	if err != nil {
		fmt.Fprintf(stderr, fmt.Sprintf("Unable to send completion message: %v\n", err))
	}
}
