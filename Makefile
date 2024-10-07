GO_CMD=go build
SRC_DIR=./cmd/beatportdl

OUTPUT_DIR=./bin

APP_NAME=beatportdl

PLATFORMS := windows/amd64 linux/amd64 darwin/amd64 darwin/arm64

all: $(PLATFORMS)

$(PLATFORMS):
	@GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) \
	$(GO_CMD) -o $(OUTPUT_DIR)/$(APP_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@)) $(SRC_DIR)
	@echo "Built $(APP_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@))"

clean:
	rm -rf $(OUTPUT_DIR)/*

.PHONY: all clean $(PLATFORMS)