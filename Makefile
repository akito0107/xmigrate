.PHONY: build
build: bin/xmigrate

.PHONY: bin/xmigrate
bin/xmigrate: vendor
	go build -o bin/xmigrate cmd/xmigrate/main.go

vendor: Gopkg.toml Gopkg.lock
	dep ensure

.PHONY: test
test: vendor
	go test ./... -cover -count=1 -v

.PHONY: install
install: vendor
	go install ./cmd/...
