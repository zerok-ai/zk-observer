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

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(NAME) cmd/main.go

buildAndPush: build
	docker build -t ${IMG_VER} .
	docker push ${IMG_VER}

ci-cd-build: sync
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-N -l" -v -o bin/$(NAME)-amd64 cmd/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -gcflags "all=-N -l" -v -o bin/$(NAME)-arm64 cmd/main.go
