package lyn

import (
	"errors"
	"fmt"
	"os"
)

const (
	elevationModeStandard = "standard"
	elevationModeAdmin    = "admin"
)

type ElevationStatus struct {
	Mode      string `json:"mode"`
	CanSwitch bool   `json:"canSwitch"`
	Message   string `json:"message,omitempty"`
}

var (
	detectElevationStatus = func() ElevationStatus {
		return ElevationStatus{Mode: elevationModeStandard, CanSwitch: false, Message: "Elevation switching is available on Windows."}
	}
	startElevationProcess = func(string, []string) error {
		return errors.New("elevation switching is available on Windows")
	}
)

func (a *App) ElevationStatus() ElevationStatus {
	return detectElevationStatus()
}

func (a *App) SwitchElevation(mode string) (ElevationStatus, error) {
	mode, err := normalizeElevationMode(mode)
	if err != nil {
		return detectElevationStatus(), err
	}
	status := detectElevationStatus()
	if status.Mode == mode {
		return status, nil
	}
	if !status.CanSwitch {
		return status, errors.New(status.Message)
	}
	ctx, _, _ := a.snapshot()
	if ctx == nil {
		return status, errors.New("application is not ready")
	}
	if err := startElevationProcess(mode, os.Args[1:]); err != nil {
		a.debugLog("elevation.switch.error", "mode", mode, "error", err)
		return status, err
	}
	a.debugLog("elevation.switch.started", "mode", mode)
	quitRuntime(ctx)
	return ElevationStatus{Mode: mode, CanSwitch: true}, nil
}

func normalizeElevationMode(mode string) (string, error) {
	switch mode {
	case elevationModeStandard, elevationModeAdmin:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported elevation mode %q", mode)
	}
}
