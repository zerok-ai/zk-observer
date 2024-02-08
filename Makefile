LOCATION = us-west1
PROJECT_ID = zerok-dev/zk-client
REPOSITORY = onlyclient01

VERSION = 0.0.3-ebpf
IMAGE = zk-observer
ART_Repo_URI = $(LOCATION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/$(IMAGE)
IMG_VER = $(ART_Repo_URI):$(VERSION)
NAME = zk-observer
BUILDER_NAME = multi-platform-builder

sync:
	go get -v ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(NAME) cmd/main.go

buildAndPush: build
	docker build -t ${IMG_VER} .
	docker push ${IMG_VER}

docker-build-push-multiarch:
	docker buildx rm ${BUILDER_NAME} || true
	docker buildx create --use --platform=linux/arm64,linux/amd64 --name ${BUILDER_NAME}
	docker buildx build --platform=linux/arm64,linux/amd64 --push --tag ${IMG_VER} .
	docker buildx rm ${BUILDER_NAME}

ci-cd-build: sync
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(NAME)-amd64 cmd/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/$(NAME)-arm64 cmd/main.go
