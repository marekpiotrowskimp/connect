// Command connect runs a single Connect instance: it opens local storage,
// loads this device's identity, and serves the web UI / local API / peer
// API over HTTP on the LAN. The tray icon is the app's lifecycle: closing
// the browser tab doesn't stop it, only quitting from the tray (or
// Ctrl+C/SIGTERM in a terminal) does.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"connect/internal/appdata"
	"connect/internal/httpapi"
	"connect/internal/identity"
	"connect/internal/storage"
	"connect/internal/tray"
	"connect/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("connect: %v", err)
	}
}

func run() error {
	dataDir, err := appdata.Dir()
	if err != nil {
		return err
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

	ln, err := httpapi.Listen(httpapi.DefaultPort)
	if err != nil {
		return err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	selfFn := func() httpapi.Device {
		ip, err := identity.LocalIP()
		if err != nil {
			ip = "127.0.0.1"
		}
		return httpapi.Device{ID: id.ID, Name: id.Name, IP: ip, Port: port}
	}

	server := httpapi.New(web.FS(), db, selfFn, appdata.FilesDir(dataDir))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := fmt.Sprintf("http://%s:%d", mustLocalIP(), port)
	log.Printf("Connect running at %s (quit from the tray icon, or Ctrl+C)", url)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Serve(ctx, ln)
	}()

	// Ctrl+C/SIGTERM should also tear down the tray, not just the server.
	go func() {
		<-ctx.Done()
		tray.Quit()
	}()

	openBrowser(url)
	tray.Run(func() { openBrowser(url) }, stop)

	return <-serverErrCh
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("could not open browser automatically, visit %s manually: %v", url, err)
	}
}

func mustLocalIP() string {
	ip, err := identity.LocalIP()
	if err != nil {
		return "127.0.0.1"
	}
	return ip
}
