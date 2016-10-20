
default: test build

SOURCE_FILES := $(shell find . -type f -name "*.go")

test:
	go test ./...

.PHONY: build
build: oci-systemd-generator

oci-systemd-generator: $(SOURCE_FILES)
	go build -o $@ .

clean:
	rm -rf *~ oci-systemd-generator

