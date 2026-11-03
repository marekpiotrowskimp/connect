//go:build !darwin

package service

import "fmt"

func installLaunchd() error {
	return fmt.Errorf("launchd service not supported on this platform")
}

func uninstallLaunchd() error {
	return fmt.Errorf("launchd service not supported on this platform")
}
