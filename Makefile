.PHONY: proto
proto:
	protoc --proto_path=. --micro_out=. --go_out=:. proto/pod/pod.proto

.PHONY: build
build:
	CGO_ENABLED="0" GOOS=linux GOARCH=amd64 go build -o pod *.go

.PHONY: docker-build
docker-build:
	docker build . -t baciyou/pod:latest

.PHONY: docker-stop
docker-stop:
	docker stop gopass-pod

.PHONY: docker-rm
docker-rm:
	docker rm gopass-pod

.PHONY: docker-rmi
docker-rmi:
	docker image rm baciyou/pod:latest

.PHONY: docker-run
docker-run:
	docker run -d --name gopass-pod -p 8081:8081 -p 9092:9092 -p 9192:9192 -v ~/.kube/config:/root/.kube/config -v /home/go-pro/src/pod/micro.log:/micro.log baciyou/pod