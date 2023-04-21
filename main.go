package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

var logFile *os.File

func main() {
	logfilePath := flag.String("logfile", "", "Path to the log file")
	flag.Parse()

	if *logfilePath != "" {
		var err error
		logFile, err = os.Create(*logfilePath)
		if err != nil {
			fmt.Println("Error creating log file:", err)
			return
		}
		defer logFile.Close()
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logError("Error loading AWS configuration:", err)
		return
	}

	s3Client := s3.NewFromConfig(cfg)

	for {
		message := &Message{}
		if err := json.NewDecoder(os.Stdin).Decode(message); err != nil {
			if err == io.EOF {
				break
			}
			logError("Error decoding JSON input:", err)
			continue
		}

		if logFile != nil {
			logMessage("Input:", message)
		}

		switch message.Event {
		case "init":
			handleInit()
		case "upload":
			handleUpload(s3Client, message.Oid, message.Size, message.Path, message.Action)
		case "download":
			handleDownload(s3Client, message.Oid, message.Size, message.Action)
		case "terminate":
			break
		}
	}
}

func handleInit() {
	response := Message{Event: "init"}
	jsonResponse, _ := json.Marshal(response)
	fmt.Println(string(jsonResponse))

	if logFile != nil {
		logMessage("Output:", string(jsonResponse))
	}
}

func handleUpload(s3Client *s3.Client, oid string, size *int64, path, action string) {
	file, err := os.Open(path)
	if err != nil {
		logError("Error opening file for upload:", err)
		return
	}
	defer file.Close()

	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
		Body:   file,
	})

	response := Message{Event: "complete", Oid: oid}
	if err != nil {
		response.Error = &Error{Code: "PutObjectError", Message: err.Error()}
	}

	jsonResponse, _ := json.Marshal(response)
	fmt.Println(string(jsonResponse))

	if logFile != nil {
		logMessage("Output:", string(jsonResponse))
	}
}

func handleDownload(s3Client *s3.Client, oid string, size *int64, action string) {
	localPath := fmt.Sprintf("%s.tmp", oid)

	file, err := os.Create(localPath)
	if err != nil {
		logError("Error creating file for download:", err)
		return
	}
	defer file.Close()

	_, err = s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(oid),
	})
	if err != nil {
		response := Message{Event: "complete", Oid: oid, Error: &Error{Code: "GetObjectError", Message: err.Error()}}
		jsonResponse, _ := json.Marshal(response)
		fmt.Println(string(jsonResponse))

		if logFile != nil {
			logMessage("Output:", string(jsonResponse))
		}
		return
	}

	_, err = io.Copy(file, resp.Body)
	resp.Body.Close()
	if err != nil {
		response := Message{Event: "complete", Oid: oid, Error: &Error{Code: "CopyError", Message: err.Error()}}
		jsonResponse, _ := json.Marshal(response)
		fmt.Println(string(jsonResponse))

		if logFile != nil {
			logMessage("Output:", string(jsonResponse))
		}
		return
	}

	response := Message{Event: "complete", Oid: oid, Path: localPath}
	jsonResponse, _ := json.Marshal(response)
	fmt.Println(string(jsonResponse))

	if logFile != nil {
		logMessage("Output:", string(jsonResponse))
	}
}

func logError(args ...interface{}) {
	if logFile != nil {
		fmt.Fprintln(logFile, args...)
	}
}

func logMessage(args ...interface{}) {
	if logFile != nil {
		fmt.Fprintln(logFile, args...)
	}
}
