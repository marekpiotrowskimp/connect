// Package web embeds the static web UI assets into the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var embedded embed.FS

// FS returns the embedded web UI, rooted at web/static so paths match
// what's served at "/" (e.g. static/index.html -> "/").
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		// Cannot happen: "static" is embedded above and always present.
		panic(err)
	}
	return sub
}
