MODULE := connect
BIN_DIR := bin
APP := $(BIN_DIR)/Connect.app
UI_DIR := ui/connect-ui

# Ensure Cargo is in PATH for Tauri builds
export PATH := $(HOME)/.cargo/bin:$(PATH)

.PHONY: run run-service build build-darwin build-windows build-connectd build-ui package package-darwin package-ui test vet clean

run:
	go run ./cmd/connect

run-service:
	go run ./cmd/connectd

run-ui:
	cd $(UI_DIR) && npx tauri dev

build: build-darwin build-windows build-connectd build-ui

# The system tray (getlantern/systray) links Cocoa on macOS via cgo, so
# CGO_ENABLED=1 must be forced explicitly: Go only defaults it on when
# GOARCH matches the host, and this cross-builds both arches from whichever
# one you're running on. Apple's clang cross-compiles the Objective-C side
# fine for either arch without extra setup.
build-darwin:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/connect-darwin-arm64 ./cmd/connect
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/connect-darwin-amd64 ./cmd/connect

# The Windows systray backend is pure Go (syscalls, no cgo), so this is a
# plain cross-compile. -H=windowsgui suppresses the console window so the
# app behaves like a tray app instead of a terminal program.
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o $(BIN_DIR)/connect-windows-amd64.exe ./cmd/connect

# V3 service binary — headless, no CGO needed on any platform.
build-connectd:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/connectd ./cmd/connectd
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/connectd-windows-amd64.exe ./cmd/connectd

# V3 desktop UI (Tauri) — requires Rust + Tauri CLI.
build-ui:
	cd $(UI_DIR)/src-tauri && cargo build --release
	cp $(UI_DIR)/src-tauri/target/release/connect-ui $(BIN_DIR)/connect-ui

build-ui-windows:
	cd $(UI_DIR)/src-tauri && cargo build --release
	cp $(UI_DIR)/src-tauri/target/release/connect-ui.exe $(BIN_DIR)/connect-ui.exe

build-ui-windows-cross:
	cd $(UI_DIR)/src-tauri && PATH="/opt/homebrew/opt/llvm/bin:$$PATH" RUSTFLAGS="-C target-feature=+crt-static" cargo xwin build --release --target x86_64-pc-windows-msvc
	cp $(UI_DIR)/src-tauri/target/x86_64-pc-windows-msvc/release/connect-ui.exe $(BIN_DIR)/connect-ui.exe

build-all-v3: build-connectd build-ui build-ui-windows-cross

# Build + package the V3 desktop app for the current platform.
package-ui: build-connectd
	cd $(UI_DIR) && npx tauri build
	cp $(BIN_DIR)/connectd $(UI_DIR)/src-tauri/target/release/connectd 2>/dev/null; true

package: package-darwin

# Bundles both darwin binaries into one universal Connect.app (LSUIElement:
# menu-bar only, no Dock icon/app switcher entry, matching a tray-only app).
package-darwin: build-darwin
	rm -rf $(APP)
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	lipo -create -output $(APP)/Contents/MacOS/connect \
		$(BIN_DIR)/connect-darwin-arm64 $(BIN_DIR)/connect-darwin-amd64
	cp packaging/darwin/Info.plist $(APP)/Contents/Info.plist
	rm -rf $(BIN_DIR)/icon.iconset
	mkdir $(BIN_DIR)/icon.iconset
	sips -z 16 16   packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_16x16.png >/dev/null
	sips -z 32 32   packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_16x16@2x.png >/dev/null
	sips -z 32 32   packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_32x32.png >/dev/null
	sips -z 64 64   packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_32x32@2x.png >/dev/null
	sips -z 128 128 packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_128x128.png >/dev/null
	sips -z 256 256 packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_128x128@2x.png >/dev/null
	sips -z 256 256 packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_256x256.png >/dev/null
	sips -z 512 512 packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_256x256@2x.png >/dev/null
	sips -z 512 512 packaging/darwin/icon-1024.png --out $(BIN_DIR)/icon.iconset/icon_512x512.png >/dev/null
	cp packaging/darwin/icon-1024.png $(BIN_DIR)/icon.iconset/icon_512x512@2x.png
	iconutil -c icns $(BIN_DIR)/icon.iconset --output $(APP)/Contents/Resources/icon.icns
	rm -rf $(BIN_DIR)/icon.iconset

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)
