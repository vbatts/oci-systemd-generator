
default: validation build

SOURCE_FILES := $(shell find . -type f -name "*.go")

.PHONY: validation
validation: test lint vet

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golint -set_exit_status ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build: oci-systemd-generator

oci-systemd-generator: $(SOURCE_FILES)
	go build -o $@ .

clean:
	rm -rf *~ oci-systemd-generator

