VERSION ?= "v1.1.0"
run:
	go run -race src/*.go

all: prep binaries docker

prep:
	mkdir -p bin

binaries: linux64 darwin64

build:
	go build src/*.go

linux64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/ps-pingdom-maintenance64 src/*.go

darwin64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/ps-pingdom-maintenanceOSX src/*.go

pack-linux64: linux64
	upx --brute bin/ps-pingdom-maintenance64

pack-darwin64: darwin64
	upx --brute bin/ps-pingdom-maintenanceOSX

docker: pack-linux64
	docker build --build-arg version="$(VERSION)" -t pasientskyhosting/ps-pingdom-maintenance:latest . && \
	docker build --build-arg version="$(VERSION)" -t pasientskyhosting/ps-pingdom-maintenance:"$(VERSION)" .

docker-run:
	docker run pasientskyhosting/ps-pingdom-maintenance:"$(VERSION)"

docker-push: docker
	docker push pasientskyhosting/ps-pingdom-maintenance:"$(VERSION)"