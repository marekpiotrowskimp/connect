// Package service manages OS-level service registration for connectd.
// It supports launchd on macOS and Windows Service on Windows.
package service

import (
	"fmt"
	"runtime"
)

// Install registers connectd as a system service.
func Install() error {
	switch runtime.GOOS {
	case "darwin":
		return installLaunchd()
	case "windows":
		return installWindows()
	default:
		return fmt.Errorf("service installation not supported on %s", runtime.GOOS)
	}
}

// Uninstall removes the system service registration.
func Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd()
	case "windows":
		return uninstallWindows()
	default:
		return fmt.Errorf("service uninstallation not supported on %s", runtime.GOOS)
	}
}
