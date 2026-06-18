package lyn

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"lyn.tools/launcher/lyn/launch"
)

type duplexConn struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (d *duplexConn) Read(p []byte) (int, error)  { return d.r.Read(p) }
func (d *duplexConn) Write(p []byte) (int, error) { return d.w.Write(p) }
func (d *duplexConn) Close() error {
	_ = d.w.Close()
	return d.r.Close()
}

func newDuplexPair() (*duplexConn, *duplexConn) {
	aRead, aWrite := io.Pipe()
	bRead, bWrite := io.Pipe()
	return &duplexConn{r: aRead, w: bWrite}, &duplexConn{r: bRead, w: aWrite}
}

func TestServeHelperExecutesAllowedRequest(t *testing.T) {
	clientEnd, serverEnd := newDuplexPair()
	var got launch.Request
	run := func(req launch.Request) launch.Result {
		got = req
		return launch.Result{Command: "app.exe"}
	}
	done := make(chan error, 1)
	go func() { done <- serveHelper(serverEnd, run) }()

	client := newHelperCodec(clientEnd)
	if err := client.writeMessage(helperRequest{ID: "7", Launch: launch.Request{Action: "run-admin", Path: `C:\app.exe`}}); err != nil {
		t.Fatal(err)
	}
	line, err := client.readLine()
	if err != nil {
		t.Fatal(err)
	}
	var resp helperResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != "7" || resp.Result.Command != "app.exe" {
		t.Fatalf("unexpected response %#v", resp)
	}
	if got.Path != `C:\app.exe` {
		t.Fatalf("helper received %#v", got)
	}
	_ = clientEnd.Close()
	if err := <-done; err != nil {
		t.Fatalf("serveHelper returned %v", err)
	}
}

func TestServeHelperRejectsNonAdminAction(t *testing.T) {
	clientEnd, serverEnd := newDuplexPair()
	called := false
	run := func(req launch.Request) launch.Result {
		called = true
		return launch.Result{}
	}
	done := make(chan error, 1)
	go func() { done <- serveHelper(serverEnd, run) }()

	client := newHelperCodec(clientEnd)
	if err := client.writeMessage(helperRequest{ID: "1", Launch: launch.Request{Action: "open", Path: `C:\app.exe`}}); err != nil {
		t.Fatal(err)
	}
	line, err := client.readLine()
	if err != nil {
		t.Fatal(err)
	}
	var resp helperResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected non-admin request to be rejected before launch")
	}
	if !strings.Contains(resp.Result.Error, "rejected") {
		t.Fatalf("unexpected response %#v", resp)
	}
	_ = clientEnd.Close()
	<-done
}

func TestServeHelperReturnsErrorOnMalformedFrame(t *testing.T) {
	clientEnd, serverEnd := newDuplexPair()
	run := func(launch.Request) launch.Result { return launch.Result{} }
	done := make(chan error, 1)
	go func() { done <- serveHelper(serverEnd, run) }()

	if _, err := clientEnd.Write([]byte("not json\n")); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil {
		t.Fatal("expected serveHelper to fail on malformed frame")
	}
	_ = clientEnd.Close()
}

func TestHelperRequestAllowed(t *testing.T) {
	cases := []struct {
		req  launch.Request
		want bool
	}{
		{launch.Request{Action: "run-admin", Path: `C:\app.exe`}, true},
		{launch.Request{Action: "RUN-ADMIN", Path: `C:\app.exe`}, true},
		{launch.Request{Action: "open", Path: `C:\app.exe`}, false},
		{launch.Request{Action: "run-admin", Path: ""}, false},
		{launch.Request{Action: "run-admin", Path: "lyn:system:shutdown"}, false},
	}
	for _, tc := range cases {
		if got := helperRequestAllowed(tc.req); got != tc.want {
			t.Fatalf("helperRequestAllowed(%#v) = %v, want %v", tc.req, got, tc.want)
		}
	}
}

func TestHelperCodecRejectsOversizedFrame(t *testing.T) {
	clientEnd, _ := newDuplexPair()
	codec := newHelperCodec(clientEnd)
	huge := strings.Repeat("a", maxHelperFrameBytes)
	if err := codec.writeMessage(helperRequest{ID: "1", Launch: launch.Request{Action: "run-admin", Path: huge}}); err == nil {
		t.Fatal("expected oversized frame to be rejected")
	}
}
