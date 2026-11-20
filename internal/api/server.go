package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const maxRequestLineBytes = 1024 * 1024

// ServeNDJSON reads newline-delimited JSON requests from r, dispatches them to
// session, and writes one JSON response line per request to w. Malformed input
// yields an error response and the loop continues until "close" or EOF.
func ServeNDJSON(r io.Reader, w io.Writer, session *Session) error {
	defer session.Close()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), maxRequestLineBytes)
	bw := bufio.NewWriterSize(w, 16*1024)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var req Request
		var resp any
		var shouldClose bool

		if err := json.Unmarshal(line, &req); err != nil {
			resp = ErrorResponse{OK: false, Error: fmt.Sprintf("invalid json: %v", err)}
		} else {
			resp, shouldClose = session.Handle(&req)
		}

		if err := enc.Encode(resp); err != nil {
			return err
		}
		if err := bw.Flush(); err != nil {
			return err
		}
		if shouldClose {
			return nil
		}
	}

	return scanner.Err()
}
