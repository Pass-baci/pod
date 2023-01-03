.PHONY: proto
proto:
	protoc --proto_path=. --micro_out=. --go_out=:. proto/pod/pod.proto

.PHONY: build
build:
	CGO_ENABLED="0" GOOS=linux GOARCH=amd64 go build -o pod *.go

.PHONY: docker
docker:
	docker build . -t baci/pod:latest