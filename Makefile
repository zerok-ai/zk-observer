LOCATION = us-west1
PROJECT_ID = zerok-dev
REPOSITORY = stage

VERSION = devclient04
IMAGE = zk-otlp-receiver
ART_Repo_URI = $(LOCATION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/$(IMAGE)
IMG_VER = $(ART_Repo_URI):$(VERSION)
NAME = zk-otlp-receiver

sync:
	go get -v ./...
zkSync:
	go clean --cache
	go clean -modcache
	go mod tidy
	go get -u github.com/zerok-ai/zk-utils-go@v0.5.0
	go mod vendor

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(NAME) cmd/main.go

buildAndPush: build
	docker build -t ${IMG_VER} .
	docker push ${IMG_VER}

ci-cd-build: sync zkSync
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-N -l" -v -o $(NAME) cmd/main.go
