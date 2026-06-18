package lyn

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"lyn.tools/launcher/lyn/launch"
)

const maxHelperFrameBytes = 64 * 1024

var (
	errHelperFrameTooLarge    = errors.New("elevated helper frame exceeds size limit")
	errHelperResponseMismatch = errors.New("elevated helper response id mismatch")
)

type launchFunc = func(launch.Request) launch.Result

type helperRequest struct {
	ID     string         `json:"id"`
	Launch launch.Request `json:"launch"`
}

type helperResponse struct {
	ID     string        `json:"id"`
	Result launch.Result `json:"result"`
}

type helperCodec struct {
	conn    io.ReadWriteCloser
	scanner *bufio.Scanner
}

func newHelperCodec(conn io.ReadWriteCloser) *helperCodec {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), maxHelperFrameBytes)
	scanner.Split(bufio.ScanLines)
	return &helperCodec{conn: conn, scanner: scanner}
}

func (c *helperCodec) readLine() ([]byte, error) {
	if c.scanner.Scan() {
		return c.scanner.Bytes(), nil
	}
	if err := c.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (c *helperCodec) close() error {
	return c.conn.Close()
}

func (c *helperCodec) writeMessage(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload)+1 > maxHelperFrameBytes {
		return errHelperFrameTooLarge
	}
	if _, err := c.conn.Write(append(payload, '\n')); err != nil {
		return err
	}
	return nil
}

func serveHelper(conn io.ReadWriteCloser, run launchFunc) error {
	codec := newHelperCodec(conn)
	for {
		line, err := codec.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		var req helperRequest
		if err := json.Unmarshal(line, &req); err != nil {
			return err
		}
		resp := helperResponse{ID: req.ID, Result: executeHelperRequest(req.Launch, run)}
		if err := codec.writeMessage(resp); err != nil {
			return err
		}
	}
}

func executeHelperRequest(req launch.Request, run launchFunc) launch.Result {
	if !helperRequestAllowed(req) {
		return launch.Result{Error: "elevated helper rejected request"}
	}
	return run(req)
}

func helperRequestAllowed(req launch.Request) bool {
	if launch.NormalizedAction(req.Action) != "run-admin" {
		return false
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return false
	}
	return !strings.HasPrefix(strings.ToLower(path), "lyn:system:")
}
