package s3adapter

import (
	"context"
	"encoding/base64"
	"hash/crc32"
	"io"
	"log"
	"math/big"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nicolas-graves/lfs-s3/compression"
	"github.com/pkg/errors"
)

type uploadTracker struct {
	reader   io.Reader
	callback func(transferred int64)
}

func (u *uploadTracker) Read(p []byte) (n int, err error) {
	n, err = u.reader.Read(p)
	if n > 0 {
		u.callback(int64(n))
	}
	return
}

func (conn *Connection) Upload(oid string, localPath string, callback func(transferred int64)) error {
	log.Printf("Received upload request for %s %s", localPath, oid)
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	remotePath := conn.asLfsPath(oid) + conn.config.Compression.Extension()
	reader, closeReader := conn.config.Compression.WrapRead(file)
	defer closeReader()

	log.Printf("Checking if file already exists")
	if ho, err := conn.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket:       aws.String(conn.config.Bucket),
		Key:          aws.String(remotePath),
		ChecksumMode: types.ChecksumModeEnabled,
	}); err == nil {
		buffer := make([]byte, 1024*256)
		var size int64
		checksummer := crc32.New(crc32.MakeTable(crc32.Castagnoli))
		for {
			n, err := reader.Read(buffer)
			if err != nil && err != io.EOF {
				return err
			}
			if n > 0 {
				size += int64(n)
				if _, err := checksummer.Write(buffer[:n]); err != nil {
					return err
				}
			}
			if err == io.EOF {
				break
			}
		}

		if ho.ContentLength != nil && *ho.ContentLength != size {
			return errors.Errorf("Existing remote file has different size, local: %d, remote: %d", size, *ho.ContentLength)
		}

		if ho.ChecksumCRC32C != nil {
			log.Printf("HeadObject checksum: %v", *ho.ChecksumCRC32C)

			rawsum := checksummer.Sum32()
			log.Printf("RawSum: 0x%x", rawsum)
			checksum := base64.StdEncoding.EncodeToString(big.NewInt(int64(rawsum)).Bytes())
			log.Printf("File checksum: %s", checksum)

			if *ho.ChecksumCRC32C != checksum {
				return errors.Errorf("Existing remote file has different checksum, local: %v, remote: %v", checksum, *ho.ChecksumCRC32C)
			}
		}

		log.Printf("File already present remotely, skipping upload")
		return nil
	}

	uploader := manager.NewUploader(conn.client, func(u *manager.Uploader) {
		u.PartSize = partSize
	})

	ut := &uploadTracker{
		reader:   reader,
		callback: callback,
	}

	log.Printf("Starting upload")
	if _, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(conn.config.Bucket),
		Key:    aws.String(remotePath),
		Body:   ut,
	}); err != nil {
		return err
	}
	log.Printf("Finished upload")

	if conn.config.DeleteOtherVersions {
		for _, c := range compression.Compressions {
			compPath := conn.asLfsPath(oid) + c.Extension()
			if compPath == remotePath {
				continue
			}
			if conn.fileExists(compPath) {
				log.Printf("Deleting other file version: %s", compPath)
				if _, err := conn.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
					Bucket: aws.String(conn.config.Bucket),
					Key:    aws.String(compPath),
				}); err != nil {
					log.Printf("Error deleting other file version: %v", err)
				}
			}
		}
	}

	return nil
}
