// Command connectd runs the Connect service (headless daemon).
// It starts the HTTP server and REST API without opening a browser or tray.
// connectd is the V3 service component; connect-ui is the desktop client.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"connect/internal/appdata"
	"connect/internal/httpapi"
	"connect/internal/identity"
	"connect/internal/service"
	"connect/internal/storage"
	"connect/web"
)

func main() {
	port := flag.Int("port", httpapi.DefaultPort, "TCP port for HTTP server")
	dataDir := flag.String("data-dir", "", "Application data directory (default: OS-specific app data)")
	daemon := flag.Bool("daemon", false, "Run in background (Unix only)")
	install := flag.Bool("install", false, "Install as system service")
	uninstall := flag.Bool("uninstall", false, "Uninstall system service")
	flag.Parse()

	if *install {
		if err := service.Install(); err != nil {
			log.Fatalf("install service: %v", err)
		}
		fmt.Println("Service installed")
		return
	}
	if *uninstall {
		if err := service.Uninstall(); err != nil {
			log.Fatalf("uninstall service: %v", err)
		}
		fmt.Println("Service uninstalled")
		return
	}

	if *daemon {
		if err := daemonize(); err != nil {
			log.Fatalf("daemonize: %v", err)
		}
		return
	}

	if err := run(*port, *dataDir); err != nil {
		log.Fatalf("connectd: %v", err)
	}
}

func run(port int, dataDir string) error {
	if dataDir == "" {
		var err error
		dataDir, err = appdata.Dir()
		if err != nil {
			return err
		}
	}
	log.Printf("data dir: %s", dataDir)

	id, err := identity.Load(appdata.IdentityPath(dataDir))
	if err != nil {
		return err
	}
	log.Printf("identity: %s (%s)", id.Name, id.ID)

	db, err := storage.Open(appdata.DBPath(dataDir))
	if err != nil {
		return err
	}
	defer db.Close()

	ln, err := httpapi.Listen(port)
	if err != nil {
		return err
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port

	selfFn := func() httpapi.Device {
		ip, err := identity.LocalIP()
		if err != nil {
			ip = "127.0.0.1"
		}
		return httpapi.Device{ID: id.ID, Name: id.Name, IP: ip, Port: actualPort}
	}

	server := httpapi.New(web.FS(), db, selfFn, appdata.FilesDir(dataDir))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := fmt.Sprintf("http://127.0.0.1:%d", actualPort)
	log.Printf("connectd running at %s", url)

	// Write port file for UI auto-discovery
	portFile := filepath.Join(dataDir, ".connectd.port")
	_ = os.WriteFile(portFile, []byte(fmt.Sprintf("%d", actualPort)), 0644)
	defer os.Remove(portFile)

	return server.Serve(ctx, ln)
}

func daemonize() error {
	args := make([]string, 0, len(os.Args))
	for _, a := range os.Args[1:] {
		if a != "--daemon" && a != "-daemon" {
			args = append(args, a)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
