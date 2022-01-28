GOARCH=amd64

.PHONY: build
build: linux windows darwin

.PHONY: linux
linux: linux-server linux-client

.PHONY: windows
windows: windows-server windows-client

.PHONY: darwin
darwin: darwin-server darwin-client

.PHONY: server
server: linux-server windows-server darwin-server

.PHONY: client
client: linux-client windows-client darwin-client

.PHONY: linux-server
linux-server:
	GOOS=linux go build -ldflags="-s -w" -o bin/server-linux-${GOARCH} ./cmd/server/main.go

.PHONY: windows-server
windows-server:
	GOOS=windows go build -ldflags="-s -w" -o bin/server-windows-${GOARCH}.exe ./cmd/server/main.go

.PHONY: darwin-server
darwin-server:
	GOOS=darwin go build -ldflags="-s -w" -o bin/server-darwin-${GOARCH} ./cmd/server/main.go

.PHONY: linux-client
linux-client:
	GOOS=linux go build -ldflags="-s -w" -o bin/client-linux-${GOARCH} ./cmd/client/main.go

.PHONY: windows-client
windows-client:
	GOOS=windows go build -ldflags="-s -w" -o bin/client-windows-${GOARCH}.exe ./cmd/client/main.go

.PHONY: darwin-client
darwin-client:
	GOOS=darwin go build -ldflags="-s -w" -o bin/client-darwin-${GOARCH} ./cmd/client/main.go

.PHONY: lint
lint:
	golangci-lint run ./cmd/... ./internal/...

.PHONY: test
test:
	go test -v -race -count=1 ./...

.PHONY: deps
deps:
	go mod verify && \
	go mod tidy

.PHONY: ca
ca:
	docker run -v "$$(pwd)/cert:/cert" cfssl/cfssl gencert -loglevel=5 -initca /cert/ca_csr.json > cert/ca.json
	cat cert/ca.json | jq -r .cert > cert/ca.crt
	cat cert/ca.json | jq -r .key > cert/ca.key
	rm cert/ca.json