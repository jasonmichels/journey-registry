build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o journey-registry .
build-docker:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o journey-registry .
	docker build -t jasonmichels/journey-registry:develop .
run:
	docker-compose build
	docker-compose up -d
test:
	go test -v ./... -bench . -cover
build-proto:
	protoc -I journey --go_out=plugins=grpc,import_path=github.com/jasonmichels/journey-registry/journey:journey journey/journey.proto