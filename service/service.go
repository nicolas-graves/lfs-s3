package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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
	tempDir := transferTempDir()
	log.Printf("Downloading into %s", tempDir)

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
			lp, err := localPath(tempDir, req.Oid)
			if err != nil {
				return err
			}
			var bytesProcessed int64
			callback := func(transferred int64) {
				bytesProcessed += transferred
				api.SendProgress(req.Oid, bytesProcessed, int(transferred), stdout, stderr)
			}
			if err := conn.Download(req.Oid, lp, callback); err != nil {
				os.Remove(lp)
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

// Downloads go to a temporary file that git-lfs moves into its object store once the transfer
// completes. The file is placed next to that store so the move stays on one volume: the LFS
// storage directory of the repository (which is shared by all worktrees and may be overridden
// with lfs.storage), or the system temporary directory outside of a repository.
func transferTempDir() string {
	if storage := lfsStorageDir(); storage != "" {
		dir := filepath.Join(storage, "tmp")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		} else {
			log.Printf("Cannot use %s for downloads: %v", dir, err)
		}
	}
	return os.TempDir()
}

func lfsStorageDir() string {
	commonDir, err := gitOutput("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		log.Printf("Not inside a git repository: %v", err)
		return ""
	}
	if storage, err := gitOutput("config", "--get", "lfs.storage"); err == nil && storage != "" {
		if filepath.IsAbs(storage) {
			return storage
		}
		return filepath.Join(commonDir, storage)
	}
	return filepath.Join(commonDir, "lfs")
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var oidRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

func localPath(dir string, oid string) (string, error) {
	if !oidRegex.MatchString(oid) {
		return "", fmt.Errorf("Invalid lfs object ID %s", oid)
	}
	return filepath.Join(dir, oid), nil
}
