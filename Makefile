GRPC_GEN_FILES=./proto/chats.proto

# Used in Dockerfile.dev for live reloading
start:
	./build/app --host 0.0.0.0 --port 80

test:
	go test -v -race -count=1 ./...

coverage:
	go test -short -count=1 -race -coverprofile=coverage.out ./...
	go tool cover -html="coverage.out"
	rm coverage.out


# Generates all grpc stuff
generate:
	protoc --go_out=. --go_opt=paths=import --go-grpc_out=. --go-grpc_opt=paths=import $(GRPC_GEN_FILES)

build: generate
	go build -o ./bin/app cmd/main.go

run: build
	docker-compose up -d


all: generate run
