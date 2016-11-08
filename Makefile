
BUILD := oci-systemd-generator
SOURCE_FILES := $(shell find . -type f -name "*.go")

default: validation build

.PHONY: validation
validation: test lint vet

.PHONY: test
test: .test

.test: $(SOURCE_FILES)
	go test -v ./... && touch $@

.PHONY: lint
lint: .lint

.lint: $(SOURCE_FILES)
	golint -set_exit_status ./... && touch $@

.PHONY: vet
vet: .vet

.vet: $(SOURCE_FILES)
	go vet ./... && touch $@

.PHONY: build
build: $(BUILD)

$(BUILD): $(SOURCE_FILES)
	go build -o $@ .

.tmpdir:
	mktemp -d > $@

run: validation .tmpdir
	go run *.go -debug $(shell cat .tmpdir) $(shell cat .tmpdir) $(shell cat .tmpdir)

clean:
	rm -rf $(shell cat .tmpdir) ; \
	rm -rf *~ $(BUILD) .lint .test .vet .tmpdir

