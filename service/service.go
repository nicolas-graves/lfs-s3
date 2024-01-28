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
	var s3Client *s3.Client
	var err error

scanner:
	for scanner.Scan() {
		line := scanner.Text()
		var req api.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(stderr, "Error reading input: %s\n", err)
			return
		}

		switch req.Event {
		case "init":
			if (os.Getenv("S3_BUCKET") == "") {
				err = fmt.Errorf("environment variable S3_BUCKET must be defined!")
				api.SendInit(1, err, writer, stderr)
			} else if (s3Client == nil) {
				s3Client, err = createS3Client()
				if err != nil {
					api.SendInit(1, err, writer, stderr)
				} else {
					api.SendInit(0, nil, writer, stderr)
				}
			} else {
				api.SendInit(0, nil, writer, stderr)
			}
		case "download":
			fmt.Fprintf(stderr, "Received download request for %s\n", req.Oid)
			retrieve(req.Oid, req.Size, writer, stderr, s3Client)
		case "upload":
			fmt.Fprintf(stderr, "Received upload request for %s\n", req.Oid)
			store(req.Oid, req.Size, writer, stderr, s3Client)
		case "terminate":
			fmt.Fprintf(stderr, "Terminating test custom adapter gracefully.\n")
			break scanner
		}
	}
}

func createS3Client() (*s3.Client, error) {
	region := os.Getenv("AWS_REGION")
	endpointURL := os.Getenv("AWS_S3_ENDPOINT")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(os.Getenv("AWS_PROFILE")),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if (endpointURL == "" || region == "") {
					// fallback to default endpoint configuration
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				} else {
					return aws.Endpoint{URL: endpointURL}, nil
				}
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		usePathStyle, err := strconv.ParseBool(os.Getenv("S3_USEPATHSTYLE"))
		if err != nil {
			usePathStyle = false
		}
		o.UsePathStyle = usePathStyle
	}), nil
}

func retrieve(oid string, size int64, writer io.Writer, stderr io.Writer, client *s3.Client) {
	bucketName := os.Getenv("S3_BUCKET")

	localPath := ".git/lfs/objects/" + oid[:2] + "/" + oid[2:4] + "/" + oid
	file, err := os.Create(localPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error creating file: %v\n", err)
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
		api.SendTransfer(oid, 1, err, localPath, writer, stderr)
	} else {
		api.SendTransfer(oid, 0, nil, localPath, writer, stderr)
	}
}

func store(oid string, size int64, writer io.Writer, stderr io.Writer, client *s3.Client) {
	bucketName := os.Getenv("S3_BUCKET")

	localPath := ".git/lfs/objects/" + oid[:2] + "/" + oid[2:4] + "/" + oid
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening file: %v\n", err)
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
		api.SendTransfer(oid, 1, err, "", writer, stderr)
	} else {
		api.SendTransfer(oid, 0, nil, "", writer, stderr)
	}
}
