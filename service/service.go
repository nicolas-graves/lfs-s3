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
	Reader         io.Reader  // only used in store
	Writer         io.WriterAt  // only used in retrieve
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

type TransferOptions struct {
	S3Client       *s3.Client
	S3Bucket       string
	S3CDN          string
	ProgressTracker *progressTracker
	LocalPath      string
}

func Serve(stdin io.Reader, stdout, stderr io.Writer) {
	scanner := bufio.NewScanner(stdin)
	writer := io.Writer(stdout)
	var err error

	transferOptions := TransferOptions{
		S3Client: nil,
		S3Bucket: os.Getenv("S3_BUCKET"),
		S3CDN:    os.Getenv("S3_BUCKET_CDN"),
		ProgressTracker: &progressTracker{
			RespWriter: writer,
			ErrWriter:  stderr,
		},
		LocalPath: "",
	}

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
			if (transferOptions.S3Bucket == "") {
				err = fmt.Errorf("environment variable S3_BUCKET must be defined!")
				api.SendInit(1, err, writer, stderr)
			} else if ((req.Operation == "upload" || transferOptions.S3CDN == "") &&
				transferOptions.S3Client == nil) {
				// s3Client doesn't need to be defined in case S3_BUCKET_CDN is.
				transferOptions.S3Client, err = createS3Client()
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
			transferOptions.ProgressTracker.Oid = req.Oid
			transferOptions.ProgressTracker.TotalSize = req.Size
			transferOptions.LocalPath = ".git/lfs/objects/" + req.Oid[:2] + "/" + req.Oid[2:4] + "/" + req.Oid
			retrieve(transferOptions)
		case "upload":
			fmt.Fprintf(stderr, "Received upload request for %s\n", req.Oid)
			transferOptions.ProgressTracker.Oid = req.Oid
			transferOptions.ProgressTracker.TotalSize = req.Size
			transferOptions.LocalPath = ".git/lfs/objects/" + req.Oid[:2] + "/" + req.Oid[2:4] + "/" + req.Oid
			store(transferOptions)
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

func retrieve(options TransferOptions) {
	file, err := os.Create(options.LocalPath)
	if err != nil {
		fmt.Fprintf(options.ProgressTracker.ErrWriter, "Error creating file: %v\n", err)
		return
	}
	defer func() {
		file.Sync()
		file.Close()
	}()

	options.ProgressTracker.Writer = &writerAtWrapper{file}

	downloader := manager.NewDownloader(options.S3Client, func(d *manager.Downloader) {
		d.PartSize = 5 * 1024 * 1024 // 1 MB part size
		d.Concurrency = 1            // Concurrent downloads
	})

	_, err = downloader.Download(context.Background(), options.ProgressTracker, &s3.GetObjectInput{
		Bucket: aws.String(options.S3Bucket),
		Key:    aws.String(options.ProgressTracker.Oid),
	})

	if err != nil {
		api.SendTransfer(options.ProgressTracker.Oid, 1, err, options.LocalPath, options.ProgressTracker.RespWriter, options.ProgressTracker.ErrWriter)
	} else {
		api.SendTransfer(options.ProgressTracker.Oid, 0, nil, options.LocalPath, options.ProgressTracker.RespWriter, options.ProgressTracker.ErrWriter)
	}
}

func store(options TransferOptions) {
	file, err := os.Open(options.LocalPath)
	if err != nil {
		fmt.Fprintf(options.ProgressTracker.ErrWriter, "Error opening file: %v\n", err)
		return
	}
	defer func() {
		file.Sync()
		file.Close()
	}()

	uploader := manager.NewUploader(options.S3Client, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024 // 1 MB part size
		// u.LeavePartsOnError = true        // Keep uploaded parts on error
	})

	options.ProgressTracker.Reader = file

	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(options.S3Bucket),
		Key:    aws.String(options.ProgressTracker.Oid),
		Body:   options.ProgressTracker,
	})

	if err != nil {
		api.SendTransfer(options.ProgressTracker.Oid, 1, err, "", options.ProgressTracker.RespWriter, options.ProgressTracker.ErrWriter)
	} else {
		api.SendTransfer(options.ProgressTracker.Oid, 0, nil, "", options.ProgressTracker.RespWriter, options.ProgressTracker.ErrWriter)
	}
}
