package lyn

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	"lyn.tools/launcher/lyn/launch"
)

type scriptedConn struct {
	requests  []helperRequest
	writeErr  error
	responder func(helperRequest) helperResponse
	pending   [][]byte
	readIdx   int
	closed    bool
}

func (c *scriptedConn) writeMessage(value any) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	req := value.(helperRequest)
	c.requests = append(c.requests, req)
	resp := helperResponse{ID: req.ID, Result: launch.Result{Command: req.Launch.Path}}
	if c.responder != nil {
		resp = c.responder(req)
	}
	line, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	c.pending = append(c.pending, line)
	return nil
}

func (c *scriptedConn) readLine() ([]byte, error) {
	if c.readIdx >= len(c.pending) {
		return nil, io.EOF
	}
	line := c.pending[c.readIdx]
	c.readIdx++
	return line, nil
}

func (c *scriptedConn) close() error {
	c.closed = true
	return nil
}

type scriptedTransport struct {
	conn       helperConn
	connectErr error
	connects   int
}

func (t *scriptedTransport) connect() (helperConn, error) {
	t.connects++
	if t.connectErr != nil {
		return nil, t.connectErr
	}
	return t.conn, nil
}

func adminRequest(path string) launch.Request {
	return launch.Request{Action: "run-admin", Path: path}
}

func TestAdminSessionReusesSingleConnection(t *testing.T) {
	conn := &scriptedConn{}
	transport := &scriptedTransport{conn: conn}
	fallbackCalls := 0
	fallback := func(launch.Request) launch.Result {
		fallbackCalls++
		return launch.Result{Error: "fallback"}
	}
	session := newAdminSession(transport, fallback, nil)

	first := session.run(adminRequest(`C:\a.exe`))
	second := session.run(adminRequest(`C:\b.exe`))

	if transport.connects != 1 {
		t.Fatalf("expected one connect, got %d", transport.connects)
	}
	if first.Command != `C:\a.exe` || second.Command != `C:\b.exe` {
		t.Fatalf("unexpected results %#v %#v", first, second)
	}
	if fallbackCalls != 0 {
		t.Fatalf("did not expect fallback, got %d calls", fallbackCalls)
	}
	if len(conn.requests) != 2 {
		t.Fatalf("expected two helper requests, got %d", len(conn.requests))
	}
}

func TestAdminSessionFallsBackWhenConnectFails(t *testing.T) {
	transport := &scriptedTransport{connectErr: errors.New("uac canceled")}
	fallback := func(launch.Request) launch.Result { return launch.Result{Command: "direct"} }
	session := newAdminSession(transport, fallback, nil)

	result := session.run(adminRequest(`C:\a.exe`))
	if result.Command != "direct" {
		t.Fatalf("expected fallback launch, got %#v", result)
	}
}

func TestAdminSessionResetsAndReconnectsAfterWriteFailure(t *testing.T) {
	conn := &scriptedConn{writeErr: errors.New("broken pipe")}
	transport := &scriptedTransport{conn: conn}
	fallbackCalls := 0
	fallback := func(launch.Request) launch.Result {
		fallbackCalls++
		return launch.Result{Command: "direct"}
	}
	session := newAdminSession(transport, fallback, nil)

	session.run(adminRequest(`C:\a.exe`))
	if !conn.closed {
		t.Fatal("expected failed connection to be closed")
	}
	session.run(adminRequest(`C:\b.exe`))
	if transport.connects != 2 {
		t.Fatalf("expected reconnect attempt, connects=%d", transport.connects)
	}
	if fallbackCalls != 2 {
		t.Fatalf("expected two fallback launches, got %d", fallbackCalls)
	}
}

func TestAdminSessionFallsBackOnResponseMismatch(t *testing.T) {
	conn := &scriptedConn{responder: func(helperRequest) helperResponse {
		return helperResponse{ID: "mismatch", Result: launch.Result{Command: "wrong"}}
	}}
	transport := &scriptedTransport{conn: conn}
	fallback := func(launch.Request) launch.Result { return launch.Result{Command: "direct"} }
	session := newAdminSession(transport, fallback, nil)

	if result := session.run(adminRequest(`C:\a.exe`)); result.Command != "direct" {
		t.Fatalf("expected fallback on id mismatch, got %#v", result)
	}
}

func TestShouldUseAdminHelper(t *testing.T) {
	if !shouldUseAdminHelper(launch.Request{Action: "run-admin"}) {
		t.Fatal("expected run-admin to use helper")
	}
	if !shouldUseAdminHelper(launch.Request{Action: "RUN-ADMIN"}) {
		t.Fatal("expected normalized run-admin to use helper")
	}
	for _, action := range []string{"open", "run-user", "code", "terminal", ""} {
		if shouldUseAdminHelper(launch.Request{Action: action}) {
			t.Fatalf("did not expect %q to use helper", action)
		}
	}
}
