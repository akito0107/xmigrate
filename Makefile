.PHONY: build
build: bin/xmigrate bin/pgmigrate bin/pginverse

.PHONY: bin/xmigrate
bin/xmigrate: vendor
	go build -o bin/xmigrate cmd/xmigrate/main.go

.PHONY: bin/pgmigrate
bin/pgmigrate: vendor
	go build -o bin/pgmigrate cmd/pgmigrate/main.go

.PHONY: bin/pginverse
bin/pginverse: vendor
	go build -o bin/pginverse cmd/pginverse/main.go

vendor: Gopkg.toml Gopkg.lock
	dep ensure

.PHONY: test
test: vendor
	go test ./... -cover -count=1 -v

.PHONY: install
install: vendor
	go install ./cmd/...

