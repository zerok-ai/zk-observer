LOCATION = us-west1
PROJECT_ID = zerok-dev
REPOSITORY = stage

VERSION = devclient04
IMAGE = zk-otlp-receiver
ART_Repo_URI = $(LOCATION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/$(IMAGE)
IMG_VER = $(ART_Repo_URI):$(VERSION)

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

sync:
	go get -v ./...

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o zk-otlp-receiver cmd/main.go

.PHONY: buildAndPush
buildAndPush: build
	docker build -t ${IMG_VER} .
	docker push ${IMG_VER}

.PHONY: install
install: kustomize
	cd k8s && $(KUSTOMIZE) edit set image zk-otlp-receiver=${IMG_VER}
	kubectl apply -k k8s

.PHONY: uninstall
uninstall: kustomize
	kubectl delete -k k8s

ci-cd-build: sync
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-N -l" -v -o $(NAME) cmd/main.go
