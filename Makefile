SHELL := /bin/bash
IMG ?= ghcr.io/m1xxos/navidrome-k8s-operator:latest
KIND_CLUSTER ?= navidrome-op

.PHONY: tidy fmt test build docker-build install-crds uninstall-crds helm-lint helm-install helm-uninstall run dev

tidy:
	go mod tidy

fmt:
	gofmt -w ./...

test:
	go test ./...

build:
	go build ./...

docker-build:
	docker build -t $(IMG) .

install-crds:
	kubectl apply -f config/crd/bases

uninstall-crds:
	kubectl delete -f config/crd/bases --ignore-not-found=true

helm-lint:
	helm lint charts/navidrome-operator

helm-install:
	helm upgrade --install navidrome-operator charts/navidrome-operator -n navidrome-operator --create-namespace

helm-uninstall:
	helm uninstall navidrome-operator -n navidrome-operator || true

run:
	go run ./main.go

dev:
	./scripts/dev-kind-helm.sh
