.PHONY: default server client deps fmt clean all release-all assets client-assets server-assets contributors
BUILDTAGS=debug

default: all

deps:	assets
	go mod download
	go mod tidy

server: deps
	mkdir -p ./bin/server
	GOOS=linux GOARCH=amd64 go build -tags '$(BUILDTAGS)' -a -ldflags="-s -w" -o ./bin/server/ngrokd cmd/ngrokd/ngrokd.go

fmt:
	go fmt ./...

client: deps
	mkdir -p ./bin/client
	GOOS=linux GOARCH=amd64 go build -tags '$(BUILDTAGS)' -a -ldflags="-s -w" -o ./bin/client/ngrok cmd/ngrok/ngrok.go

assets: client-assets server-assets

bin/go-bindata:
	go install github.com/go-bindata/go-bindata/go-bindata@latest

client-assets: bin/go-bindata
	go-bindata -nomemcopy -pkg=assets -tags=$(BUILDTAGS) \
		-debug=$(if $(findstring debug,$(BUILDTAGS)),true,false) \
		-o=pkg/client/assets/assets_$(BUILDTAGS).go \
		assets/client/...

server-assets: bin/go-bindata
	npm run build-css-prod
	go-bindata -nomemcopy -pkg=assets -tags=$(BUILDTAGS) -ignore \.css.i$ \
		-debug=$(if $(findstring debug,$(BUILDTAGS)),true,false) \
		-o=pkg/server/assets/assets_$(BUILDTAGS).go \
		assets/server/...

release-client: BUILDTAGS=release
release-client: client

release-server: BUILDTAGS=release
release-server: server-assets server

release-all: fmt release-client release-server


all: client server

clean:
	rm -rf pkg/client/assets/ pkg/server/assets/

contributors:
	echo "Contributors to ngrok, both large and small:\n" > CONTRIBUTORS
	git log --raw | grep "^Author: " | sort | uniq | cut -d ' ' -f2- | sed 's/^/- /' | cut -d '<' -f1 >> CONTRIBUTORS
