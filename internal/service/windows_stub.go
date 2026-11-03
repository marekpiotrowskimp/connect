//go:build !windows

package service

import "fmt"

func installWindows() error {
	return fmt.Errorf("Windows service not supported on this platform")
}

func uninstallWindows() error {
	return fmt.Errorf("Windows service not supported on this platform")
}
