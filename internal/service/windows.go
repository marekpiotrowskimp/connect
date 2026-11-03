//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
)

const serviceName = "connectd"

func installWindows() error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command("sc", "create", serviceName,
		"binPath=", binary,
		"start=", "auto",
		"DisplayName=", "Connect Service",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc create: %w\n%s", err, out)
	}
	return nil
}

func uninstallWindows() error {
	cmd := exec.Command("sc", "delete", serviceName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sc delete: %w\n%s", err, out)
	}
	return nil
}
