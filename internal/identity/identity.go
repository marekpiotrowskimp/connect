// Package identity manages this Connect instance's persistent identity
// (a stable unique ID and a human-readable display name) and exposes a
// helper to detect the machine's LAN IP address.
package identity

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"connect/internal/idgen"
)

// Identity is the persisted identity of this Connect instance.
type Identity struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	path string
}

// Load reads the identity from path, creating and persisting a new one
// (random ID, name defaulting to the OS hostname) if it doesn't exist yet.
func Load(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("identity: parse %s: %w", path, err)
		}
		id.path = path
		return &id, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("identity: read %s: %w", path, err)
	}

	id := &Identity{ID: idgen.New(), Name: defaultName(), path: path}
	if err := id.Save(); err != nil {
		return nil, err
	}
	return id, nil
}

// Save persists the identity to its original path.
func (id *Identity) Save() error {
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return fmt.Errorf("identity: marshal: %w", err)
	}
	if err := os.WriteFile(id.path, data, 0o644); err != nil {
		return fmt.Errorf("identity: write %s: %w", id.path, err)
	}
	return nil
}

// SetName updates the display name and persists it.
func (id *Identity) SetName(name string) error {
	id.Name = name
	return id.Save()
}

func defaultName() string {
	if host, err := os.Hostname(); err == nil && host != "" {
		return host
	}
	return "Connect Device"
}

// LocalIP returns this machine's primary LAN IPv4 address. It first tries
// the UDP-dial heuristic (no actual traffic sent), then falls back to
// enumerating network interfaces for a non-loopback IPv4 address.
func LocalIP() (string, error) {
	ip, err := localIPViaDial()
	if err == nil && ip != "" {
		return ip, nil
	}
	ip, err = localIPViaInterfaces()
	if err == nil && ip != "" {
		return ip, nil
	}
	return "", fmt.Errorf("identity: detect local ip: %w", err)
}

func localIPViaDial() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil || addr.IP.IsLoopback() {
		return "", fmt.Errorf("unexpected local addr %T %v", conn.LocalAddr(), addr)
	}
	return addr.IP.String(), nil
}

func localIPViaInterfaces() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		return ipnet.IP.String(), nil
	}
	return "", fmt.Errorf("no non-loopback IPv4 interface found")
}
