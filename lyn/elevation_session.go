package lyn

import (
	"encoding/json"
	"strconv"
	"sync"

	"lyn.tools/launcher/lyn/launch"
)

type helperTransport interface {
	connect() (helperConn, error)
}

type helperConn interface {
	readLine() ([]byte, error)
	writeMessage(value any) error
	close() error
}

type adminSession struct {
	mu        sync.Mutex
	transport helperTransport
	conn      helperConn
	counter   uint64
	fallback  launchFunc
	log       func(stage string, values ...any)
}

func newAdminSession(transport helperTransport, fallback launchFunc, log func(string, ...any)) *adminSession {
	return &adminSession{transport: transport, fallback: fallback, log: log}
}

func (s *adminSession) run(req launch.Request) launch.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.exchange(req)
	if err != nil {
		s.logf("elevation.helper.error", "error", err)
		s.resetLocked()
		return s.fallback(req)
	}
	return result
}

func (s *adminSession) exchange(req launch.Request) (launch.Result, error) {
	if s.conn == nil {
		conn, err := s.transport.connect()
		if err != nil {
			return launch.Result{}, err
		}
		s.conn = conn
		s.logf("elevation.helper.started")
	}
	s.counter++
	id := strconv.FormatUint(s.counter, 10)
	if err := s.conn.writeMessage(helperRequest{ID: id, Launch: req}); err != nil {
		return launch.Result{}, err
	}
	line, err := s.conn.readLine()
	if err != nil {
		return launch.Result{}, err
	}
	var resp helperResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return launch.Result{}, err
	}
	if resp.ID != id {
		return launch.Result{}, errHelperResponseMismatch
	}
	return resp.Result, nil
}

func (s *adminSession) resetLocked() {
	if s.conn != nil {
		_ = s.conn.close()
		s.conn = nil
	}
}

func (s *adminSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetLocked()
}

func (s *adminSession) logf(stage string, values ...any) {
	if s.log != nil {
		s.log(stage, values...)
	}
}

func shouldUseAdminHelper(req launch.Request) bool {
	return launch.NormalizedAction(req.Action) == "run-admin"
}
