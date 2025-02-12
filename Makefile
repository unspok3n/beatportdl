ifneq (,$(wildcard ./.env))
    include .env
    export
endif

BUILD_CMD = go build -ldflags "-w -linkmode external -extldflags '-lstdc++'" -buildmode pie
BUILD_SRC = ./cmd/beatportdl
BUILD_DIR = ./bin

ZIG_CC = zig cc
ZIG_CXX = zig c++

MACOS_SDK_PATH ?= /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk

all: darwin-arm64 darwin-amd64 linux-amd64 linux-arm64 windows-amd64

darwin-arm64:
	@echo "Building for macOS ARM64"
	go clean -cache
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=arm64 \
	CGO_LDFLAGS="-F${MACOS_SDK_PATH}/System/Library/Frameworks -L${MACOS_SDK_PATH}/usr/lib" \
	CC="${ZIG_CC} -target aarch64-macos ${MACOS_ARM64_LIB_PATH} -isysroot ${MACOS_SDK_PATH} -iwithsysroot /usr/include -iframeworkwithsysroot /System/Library/Frameworks" \
	CXX="${ZIG_CXX} -target aarch64-macos ${MACOS_ARM64_LIB_PATH} -isysroot ${MACOS_SDK_PATH} -iwithsysroot /usr/include -iframeworkwithsysroot /System/Library/Frameworks" \
	${BUILD_CMD} -o=${BUILD_DIR}/beatportdl-darwin-arm64 ${BUILD_SRC}

darwin-amd64:
	@echo "Building for macOS AMD64"
	go clean -cache
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=amd64 \
	CGO_LDFLAGS="-F${MACOS_SDK_PATH}/System/Library/Frameworks -L${MACOS_SDK_PATH}/usr/lib" \
	CC="${ZIG_CC} -target x86_64-macos ${MACOS_AMD64_LIB_PATH} -isysroot ${MACOS_SDK_PATH} -iwithsysroot /usr/include -iframeworkwithsysroot /System/Library/Frameworks" \
	CXX="${ZIG_CXX} -target x86_64-macos ${MACOS_AMD64_LIB_PATH} -isysroot ${MACOS_SDK_PATH} -iwithsysroot /usr/include -iframeworkwithsysroot /System/Library/Frameworks" \
	${BUILD_CMD} -o=${BUILD_DIR}/beatportdl-darwin-amd64 ${BUILD_SRC}

linux-amd64:
	@echo "Building for Linux AMD64"
	go clean -cache
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC="${ZIG_CC} -target x86_64-linux-gnu ${LINUX_AMD64_LIB_PATH} -DTAGLIB_STATIC -Wall" \
	CXX="${ZIG_CXX} -target x86_64-linux-gnu ${LINUX_AMD64_LIB_PATH} -DTAGLIB_STATIC -Wall" \
	${BUILD_CMD} -o=${BUILD_DIR}/beatportdl-linux-amd64 ${BUILD_SRC}

linux-arm64:
	@echo "Building for Linux ARM64"
	go clean -cache
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=arm64 \
	CC="${ZIG_CC} -target aarch64-linux-gnu ${LINUX_ARM64_LIB_PATH} -DTAGLIB_STATIC -Wall" \
	CXX="${ZIG_CXX} -target aarch64-linux-gnu ${LINUX_ARM64_LIB_PATH} -DTAGLIB_STATIC -Wall" \
	${BUILD_CMD} -o=${BUILD_DIR}/beatportdl-linux-arm64 ${BUILD_SRC}

windows-amd64:
	@echo "Building for Windows AMD64"
	go clean -cache
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	CC="${ZIG_CC} -target x86_64-windows-gnu ${WINDOWS_AMD64_LIB_PATH} -DTAGLIB_STATIC -Wall -Wno-deprecated" \
	CXX="${ZIG_CXX} -target x86_64-windows-gnu ${WINDOWS_AMD64_LIB_PATH} -DTAGLIB_STATIC -Wall -Wno-deprecated" \
	${BUILD_CMD} -o=${BUILD_DIR}/beatportdl-windows-amd64.exe ${BUILD_SRC}