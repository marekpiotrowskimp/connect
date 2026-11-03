// Package appdata resolves the per-OS application data directory used to
// store the SQLite database, received files, and instance identity.
package appdata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const appName = "Connect"
const storageGeneration = "paired-workspace-v1"

// Dir returns the root application data directory for the current OS,
// creating it (and the "files" subdirectory) if it doesn't exist yet.
//
//   - macOS:   ~/Library/Application Support/Connect
//   - Windows: %APPDATA%\Connect
//   - other:   $XDG_DATA_HOME/Connect or ~/.local/share/Connect
func Dir() (string, error) {
	root, err := rootDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(root, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("appdata: create dir: %w", err)
	}
	// This is a deliberate product reset, not a database migration. Old
	// connection workspaces and relay files cannot be interpreted as a shared
	// paired workspace, so remove them once before starting the new model.
	marker := filepath.Join(dir, storageGeneration)
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		if err := os.Remove(DBPath(dir)); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("appdata: reset database: %w", err)
		}
		if err := os.RemoveAll(FilesDir(dir)); err != nil {
			return "", fmt.Errorf("appdata: reset files: %w", err)
		}
		if err := os.WriteFile(marker, nil, 0o644); err != nil {
			return "", fmt.Errorf("appdata: write reset marker: %w", err)
		}
	}
	if err := os.MkdirAll(FilesDir(dir), 0o755); err != nil {
		return "", fmt.Errorf("appdata: create files dir: %w", err)
	}
	return dir, nil
}

// FilesDir returns the directory under root where received files are stored.
func FilesDir(root string) string {
	return filepath.Join(root, "files")
}

// DBPath returns the path to the SQLite database file under root.
func DBPath(root string) string {
	return filepath.Join(root, "connect.db")
}

// IdentityPath returns the path to the persisted identity JSON file under root.
func IdentityPath(root string) string {
	return filepath.Join(root, "identity.json")
}

func rootDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("appdata: user home dir: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("appdata: user home dir: %w", err)
		}
		return filepath.Join(home, "AppData", "Roaming"), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return xdg, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("appdata: user home dir: %w", err)
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}
