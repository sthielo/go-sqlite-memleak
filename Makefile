
NAME       		:= oom

ifeq ($(OS),Windows_NT)
	NAME := oom.exe
endif

MAIN_SRC   		:= pkg/internal/main.go
BUILD_DIR  		:= artifacts/
BINARY     		:= $(BUILD_DIR)$(NAME)
#GOOS       	:= linux    # leave this to self detection, so it may work locally
GOARCH     		:= amd64
GO_TEST_FLAGS	?= -gcflags="all=-N -l"
TEMP_DIR		:= tmp/

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(TEMP_DIR)

deps:
	go mod tidy
	go mod download

build: clean deps
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_FLAGS="-DSQLITE_USE_URI=1 -DSQLITE_THREADSAFE=0" go build -tags=nomsgpack -o $(BINARY) $(MAIN_SRC)

lint-all:

test: build
	mkdir -p $(TEMP_DIR)
	echo "WARNING: running several minutes ..."
	GIN_MODE=release go test $(GO_TEST_FLAGS) -v -timeout 40m ./httptesting 2>&1

test-all: test

run: build
	mkdir -p $(TEMP_DIR)
	./$(BINARY)
