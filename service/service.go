package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"

	"github.com/nicolas-graves/lfs-s3/api"
	"github.com/nicolas-graves/lfs-s3/s3adapter"
)

func Serve(stdin io.Reader, stdout, stderr io.Writer, config *s3adapter.Config) error {
	if config.Bucket == "" {
		return fmt.Errorf("no bucket set")
	}
	if config.Endpoint == "" {
		return fmt.Errorf("no endpoint set")
	}
	if config.Compression == nil {
		return fmt.Errorf("invalid compression set")
	}
	if (config.AccessKeyId == "") != (config.SecretAccessKey == "") {
		return fmt.Errorf("access key and secret key should either both be set or both be empty")
	}

	conn, err := s3adapter.New(config)
	if err != nil {
		return err
	}
	log.Printf("Serving LFS")

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("Read line %s", line)
		var req api.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return fmt.Errorf("error reading input: %s", err)
		}
		log.Printf("Received request %+v", req)
		switch req.Event {
		case "init":
			api.SendInit(0, nil, stdout, stderr)
		case "terminate":
			log.Printf("Terminating test custom adapter gracefully.")
		case "download":
			lp, err := localPath(req.Oid)
			if err != nil {
				return err
			}
			var bytesProcessed int64
			callback := func(transferred int64) {
				bytesProcessed += transferred
				api.SendProgress(req.Oid, bytesProcessed, int(transferred), stdout, stderr)
			}
			if err := conn.Download(req.Oid, lp, callback); err != nil {
				api.SendTransfer(req.Oid, 1, err, lp, stdout, stderr)
			} else {
				api.SendTransfer(req.Oid, 0, nil, lp, stdout, stderr)
			}
		case "upload":
			var bytesProcessed int64
			callback := func(transferred int64) {
				bytesProcessed += transferred
				api.SendProgress(req.Oid, bytesProcessed, int(transferred), stdout, stderr)
			}

			if err := conn.Upload(req.Oid, req.Path, callback); err != nil {
				api.SendTransfer(req.Oid, 1, err, "", stdout, stderr)
			} else {
				api.SendTransfer(req.Oid, 0, nil, "", stdout, stderr)
			}
		default:
			log.Printf("Unknown event: %s", req.Event)
		}
	}
	return nil
}

var oidRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

func localPath(oid string) (string, error) {
	if !oidRegex.MatchString(oid) {
		return "", fmt.Errorf("Invalid lfs object ID %s", oid)
	}
	return fmt.Sprintf(".git/lfs/objects/%s/%s/%s", oid[:2], oid[2:4], oid), nil
}
