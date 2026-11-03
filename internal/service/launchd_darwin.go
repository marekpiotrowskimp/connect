//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const plistName = "local.connect.connectd.plist"

func launchdDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func binaryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return execPath, nil
}

func installLaunchd() error {
	dir, err := launchdDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	binary, err := binaryPath()
	if err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>local.connect.connectd</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ThrottleInterval</key>
	<integer>5</integer>
	<key>StandardOutPath</key>
	<string>/tmp/connectd.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/connectd.log</string>
</dict>
</plist>
`, binary)

	path := filepath.Join(dir, plistName)
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return err
	}

	return exec.Command("launchctl", "load", path).Run()
}

func uninstallLaunchd() error {
	dir, err := launchdDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, plistName)

	// Unload first, ignore error if not loaded
	_ = exec.Command("launchctl", "unload", path).Run()

	return os.Remove(path)
}
