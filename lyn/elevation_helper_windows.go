//go:build windows

package lyn

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"

	"lyn.tools/launcher/lyn/launch"
)

const (
	pipeAccessDuplex       = 0x00000003
	pipeModeByteStream     = 0x00000000
	pipeUnlimitedInstances = 255
	pipeBufferSize         = 4096
	helperConnectTimeout   = 30 * time.Second
)

var errHelperConnectTimeout = errors.New("timed out waiting for elevated helper to connect")

func newWindowsAdminLaunchSession(fallback launchFunc, log func(string, ...any)) *adminSession {
	if windows.GetCurrentProcessToken().IsElevated() {
		return nil
	}
	return newAdminSession(&windowsHelperTransport{log: log}, fallback, log)
}

type windowsHelperTransport struct {
	log func(string, ...any)
}

func (t *windowsHelperTransport) connect() (helperConn, error) {
	name := newHelperPipeName()
	server, err := createHelperPipeServer(name)
	if err != nil {
		return nil, err
	}
	if err := startElevatedHelper(name); err != nil {
		_ = server.Close()
		return nil, err
	}
	if err := server.accept(helperConnectTimeout); err != nil {
		_ = server.Close()
		return nil, err
	}
	return newHelperCodec(server), nil
}

func runWindowsElevatedHelper(pipeName string) error {
	conn, err := dialHelperPipe(pipeName)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	return serveHelper(conn, runElevatedTarget)
}

func runElevatedTarget(req launch.Request) launch.Result {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return launch.Result{Error: "launch path is required"}
	}
	if err := shellExecuteProcess("open", path, "", "", windowsShellShowNormal); err != nil {
		return launch.Result{Command: path, Error: err.Error()}
	}
	return launch.Result{Command: path}
}

func newHelperPipeName() string {
	return `\\.\pipe\lyn-elevated-` + uuid.NewString()
}

func startElevatedHelper(pipeName string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return shellExecuteProcess("runas", exe, "--elevated-helper="+pipeName, filepath.Dir(exe), windowsShellHide)
}

type serverPipe struct {
	handle windows.Handle
}

func createHelperPipeServer(name string) (*serverPipe, error) {
	sa, err := helperPipeSecurityAttributes()
	if err != nil {
		return nil, err
	}
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateNamedPipe(
		namePtr,
		pipeAccessDuplex,
		pipeModeByteStream,
		pipeUnlimitedInstances,
		pipeBufferSize,
		pipeBufferSize,
		0,
		sa,
	)
	if err != nil {
		return nil, err
	}
	return &serverPipe{handle: handle}, nil
}

func (s *serverPipe) accept(timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		err := windows.ConnectNamedPipe(s.handle, nil)
		if err == windows.ERROR_PIPE_CONNECTED {
			err = nil
		}
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = windows.CancelIoEx(s.handle, nil)
		<-done
		return errHelperConnectTimeout
	}
}

func (s *serverPipe) Read(p []byte) (int, error) {
	return readPipe(s.handle, p)
}

func (s *serverPipe) Write(p []byte) (int, error) {
	return writePipe(s.handle, p)
}

func (s *serverPipe) Close() error {
	_ = windows.DisconnectNamedPipe(s.handle)
	return windows.CloseHandle(s.handle)
}

type clientPipe struct {
	handle windows.Handle
}

func dialHelperPipe(name string) (io.ReadWriteCloser, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		namePtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, err
	}
	return &clientPipe{handle: handle}, nil
}

func (c *clientPipe) Read(p []byte) (int, error) {
	return readPipe(c.handle, p)
}

func (c *clientPipe) Write(p []byte) (int, error) {
	return writePipe(c.handle, p)
}

func (c *clientPipe) Close() error {
	return windows.CloseHandle(c.handle)
}

func readPipe(handle windows.Handle, p []byte) (int, error) {
	var read uint32
	if err := windows.ReadFile(handle, p, &read, nil); err != nil {
		if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED || err == windows.ERROR_NO_DATA {
			return int(read), io.EOF
		}
		return int(read), err
	}
	return int(read), nil
}

func writePipe(handle windows.Handle, p []byte) (int, error) {
	var written uint32
	if err := windows.WriteFile(handle, p, &written, nil); err != nil {
		return int(written), err
	}
	return int(written), nil
}

func helperPipeSecurityAttributes() (*windows.SecurityAttributes, error) {
	sddl, err := currentUserPipeSDDL()
	if err != nil {
		return nil, err
	}
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, err
	}
	sa := &windows.SecurityAttributes{}
	sa.Length = uint32(unsafe.Sizeof(*sa))
	sa.SecurityDescriptor = descriptor
	return sa, nil
}

func currentUserPipeSDDL() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return "D:P(A;;GA;;;" + user.User.Sid.String() + ")", nil
}
