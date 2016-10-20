
default: validation build

SOURCE_FILES := $(shell find . -type f -name "*.go")

.PHONY: validation
validation: test lint vet

.PHONY: test
test: .test

.test: $(SOURCE_FILES)
	go test ./... && touch $@

.PHONY: lint
lint: .lint

.lint: $(SOURCE_FILES)
	golint -set_exit_status ./... && touch $@

.PHONY: vet
vet: .vet

.vet: $(SOURCE_FILES)
	go vet ./... && touch $@

.PHONY: build
build: oci-systemd-generator

oci-systemd-generator: $(SOURCE_FILES)
	go build -o $@ .

clean:
	rm -rf *~ oci-systemd-generator .lint .test .vet

