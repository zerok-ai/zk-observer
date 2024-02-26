NAME = zk-observer

# VERSION defines the project version for the project.
# Update this value when you upgrade the version of your project.
IMAGE_VERSION ?= latest

#Docker image location
#change this to your docker hub username
DOCKER_HUB ?= zerokai
IMAGE_NAME ?= zk-observer
ART_Repo_URI ?= $(DOCKER_HUB)/$(IMAGE_NAME)
IMG ?= $(ART_Repo_URI):$(IMAGE_VERSION)


sync:
	go get -v ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(NAME) cmd/main.go

buildAndPush: build
	docker build -t ${IMG} .
	docker push ${IMG}

docker-build-push-multiarch:
	docker buildx rm ${BUILDER_NAME} || true
	docker buildx create --use --platform=linux/arm64,linux/amd64 --name ${BUILDER_NAME}
	docker buildx build --platform=linux/arm64,linux/amd64 --push --tag ${IMG_VER} .
	docker buildx rm ${BUILDER_NAME}

ci-cd-build: sync
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(NAME)-amd64 cmd/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/$(NAME)-arm64 cmd/main.go
