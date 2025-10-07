package s3adapter

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nicolas-graves/lfs-s3/compression"
	"github.com/pkg/errors"
)

type writerAtWrapper struct {
	w   io.Writer
	pos int64
}

func (waw *writerAtWrapper) WriteAt(p []byte, off int64) (n int, err error) {
	if off != waw.pos {
		return 0, errors.Errorf("Invalid WriteAt position, expected: %d, got: %d", waw.pos, off)
	}
	waw.pos += int64(len(p))
	return waw.w.Write(p)
}

type downloadTracker struct {
	writer   io.WriterAt
	callback func(transferred int64)
}

func (dt *downloadTracker) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = dt.writer.WriteAt(p, off)
	if n > 0 {
		dt.callback(int64(n))
	}
	return
}

func (conn *Connection) Download(oid string, localPath string, callback func(transferred int64)) error {
	log.Printf("Received download request for %s", oid)
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var comp compression.Compression
	basePath := conn.asLfsPath(oid)

	for _, c := range compression.Compressions {
		compPath := basePath + c.Extension()
		log.Printf("Checking %s", compPath)
		if conn.fileExists(compPath) {
			comp = c
			basePath = compPath
			break
		}
	}

	if comp == nil {
		return errors.Errorf("No downloadable version of the file was found")
	}

	log.Printf("Resolved remote path: %s with compression %s", basePath, comp.Name())

	writer, closeWriter := comp.WrapWrite(file)
	defer closeWriter()
	dt := &downloadTracker{
		writer:   &writerAtWrapper{w: writer},
		callback: callback,
	}

	downloader := manager.NewDownloader(conn.client, func(d *manager.Downloader) {
		d.PartSize = partSize
		d.Concurrency = 1
	})
	if _, err := downloader.Download(context.Background(), dt, &s3.GetObjectInput{
		Bucket: aws.String(conn.config.Bucket),
		Key:    aws.String(basePath),
	}); err != nil {
		return err
	}

	log.Printf("Download of %s finished, returning", basePath)
	return nil
}
