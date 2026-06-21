# Image Transmit Build Makefile
# Supports building CLI, Windows GUI, and Linux GUI binaries

APP_NAME       := image-transmit
WIN_GUI_DIR    := ./win
LINUX_GUI_DIR  := ./linuxgui
CMD_DIR        := ./cmd
BUILD_DIR      := ./build
EMBEDDED_DIR   := ./embedded
SCRIPTS_DIR    := ./scripts

LDFLAGS        := -s -w
WIN_GUI_FLAGS  := -ldflags "$(LDFLAGS) -H windowsgui"
CLI_FLAGS      := -ldflags "$(LDFLAGS)"
CLI_EMBED_FLAGS := -ldflags "$(LDFLAGS)" -tags embedded_tools

EMBEDDED_TOOLS := skopeo ctr crictl nerdctl regctl mc redis-cli

.PHONY: all
all: test build-cli build-win build-linux-gui

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: test
test:
	go test ./core/... ./cmd/... ./linuxgui/...

.PHONY: build-cli
build-cli: $(BUILD_DIR)
	go build $(CLI_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

.PHONY: build-cli-full
build-cli-full: download-tools $(BUILD_DIR)
	go build $(CLI_EMBED_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Creating symlinks for embedded tools..."
	@cd $(BUILD_DIR) && for tool in $(EMBEDDED_TOOLS); do ln -sf $(APP_NAME) $$tool; done

.PHONY: build-win
build-win: $(BUILD_DIR)
	cd $(WIN_GUI_DIR) && GOOS=windows GOARCH=amd64 go build $(WIN_GUI_FLAGS) -o ../$(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe

.PHONY: build-win-x86
build-win-x86: $(BUILD_DIR)
	cd $(WIN_GUI_DIR) && GOOS=windows GOARCH=386 go build $(WIN_GUI_FLAGS) -o ../$(BUILD_DIR)/$(APP_NAME)-windows-x86.exe

.PHONY: build-linux-gui
build-linux-gui: $(BUILD_DIR)
	cd $(LINUX_GUI_DIR) && GOOS=linux GOARCH=amd64 go build $(CLI_FLAGS) -o ../$(BUILD_DIR)/$(APP_NAME)-linux-gui

.PHONY: build-all
build-all: test build-cli build-win build-win-x86 build-linux-gui

.PHONY: download-tools
download-tools:
	@bash $(SCRIPTS_DIR)/download-tools.sh

.PHONY: symlinks
symlinks:
	@cd $(BUILD_DIR) && for tool in $(EMBEDDED_TOOLS); do ln -sf $(APP_NAME) $$tool; done

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: clean-tools
clean-tools:
	rm -rf $(EMBEDDED_DIR)/bin

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./core/... ./cmd/... ./embedded/...
