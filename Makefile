## Makefile for building SimplePresent (Flutter)
#
# Usage:
#  make build-linux    # Build Linux release (must run on Linux host with Flutter desktop enabled)
#  make build-windows  # Build Windows release (must run on Windows host with Flutter desktop enabled)
#  make build-all      # Build both (runs both targets; Windows build will fail on non-Windows hosts)
#  make package-linux  # Copy Linux bundle to dist/
#  make package-windows# Copy Windows exe to dist/ (Windows host)
#  make clean
#  make pub-get
#  make test

FLUTTER ?= flutter
DIST_DIR := dist

.PHONY: all build-linux build-windows build-all package-linux package-windows clean pub-get test

all: build-linux

build-linux:
	@echo "==> ensure flutter packages"
	$(FLUTTER) pub get
	@echo "==> building Linux (release)"
	$(FLUTTER) build linux --release

build-windows:
	@echo "==> ensure flutter packages"
	$(FLUTTER) pub get
	@echo "==> building Windows (release)"
	@echo "Note: Building Windows requires a Windows host or a cross-compilation workflow."
	$(FLUTTER) build windows --release

build-all: build-linux build-windows

package-linux: build-linux
	@mkdir -p $(DIST_DIR)
	@echo "==> copying linux bundle to $(DIST_DIR)"
	@cp -r build/linux/runner/release/bundle $(DIST_DIR)/simplepresent-linux || true

package-windows: build-windows
	@mkdir -p $(DIST_DIR)
	@echo "==> copying windows exe to $(DIST_DIR)"
	@cp -r build/windows/runner/Release/simplepresent.exe $(DIST_DIR)/simplepresent-windows.exe || true

clean:
	$(FLUTTER) clean

pub-get:
	$(FLUTTER) pub get

test:
	$(FLUTTER) test
