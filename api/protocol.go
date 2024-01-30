package api

import (
	"encoding/json"
	"fmt"
	"io"
)

// The protocol aims to be a suitable implementation of
// https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md,
// nothing more, nothing less.

// Header struct
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Error struct
type Message struct {
	Event  string  `json:"event"`
	Oid    string  `json:"oid"`
	Size   *int64  `json:"size,omitempty"`
	Path   string  `json:"path,omitempty"`
	Action string  `json:"action,omitempty"`
	Error  *Error  `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Request struct which can accept anything
type Request struct {
	Event               string  `json:"event"`
	Operation           string  `json:"operation"`
	Concurrent          bool    `json:"concurrent"`
	ConcurrentTransfers int     `json:"concurrenttransfers"`
	Oid                 string  `json:"oid"`
	Size                int64   `json:"size"`
	Path                string  `json:"path"`
}

// InitResponse with response for init
type InitResponse struct {
	Error *Error `json:"error,omitempty"`
}

// TransferResponse generic transfer response
type TransferResponse struct {
	Event string         `json:"event"`
	Oid   string         `json:"oid"`
	Path  string         `json:"path,omitempty"` // always blank for upload
	Error *Error `json:"error,omitempty"`
}

// ProgressResponse blah
type ProgressResponse struct {
	Event          string `json:"event"`
	Oid            string `json:"oid"`
	BytesSoFar     int64  `json:"bytesSoFar"`
	BytesSinceLast int    `json:"bytesSinceLast"`
}

// SendResponse sends an actual response to lfs
func SendResponse(r interface{}, writer io.Writer, stderr io.Writer) error {
	b, err := json.Marshal(r)
	if err != nil {
		fmt.Fprintf(stderr, "Error marshalling response: %s", err)
		return err
	}
	// Line oriented JSON
	b = append(b, '\n')
	_, err = writer.Write(b)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Sent message %v", string(b))
	return nil
}

// SendInit answers to init
func SendInit(code int, err error, writer io.Writer, stderr io.Writer) {
	var resp *InitResponse
	if err != nil {
		resp = &InitResponse{&Error{code, fmt.Sprintf("Init error: %s\n", err)}}
	} else {
		resp = &InitResponse{}
	}
	respErr := SendResponse(resp, writer, stderr)
	if respErr != nil {
		fmt.Fprintf(stderr, "Unable to send init response: %v\n", respErr)
	}
}

// SendTransfer sends a transfer message back to lfs
func SendTransfer(oid string, code int, err error, path string, writer io.Writer, stderr io.Writer) {
	var resp *TransferResponse
	if err != nil {
		var message string
		if path == "" {  // always empty on upload
			message = fmt.Sprintf("Error uploading file: %v\n", err)
		} else {
			message = fmt.Sprintf("Error downloading file: %v\n", err)
		}
		resp = &TransferResponse{"complete", oid, "", &Error{code, message}}
	} else {
		if path == "" {
			resp = &TransferResponse{Event: "complete", Oid: oid, Error: nil}
		} else {
			resp = &TransferResponse{Event: "complete", Oid: oid, Path: path, Error: nil}
		}
	}
	respErr := SendResponse(resp, writer, stderr)
	if respErr != nil {
		fmt.Fprintf(stderr, "Unable to send transfer message: %v\n", respErr)
	}
}

// SendProgress reports progress on operations
func SendProgress(oid string, bytesSoFar int64, bytesSinceLast int, writer io.Writer, stderr io.Writer) {
	resp := &ProgressResponse{"progress", oid, bytesSoFar, bytesSinceLast}
	err := SendResponse(resp, writer, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Unable to send progress update: %v\n", err)
	}
}
