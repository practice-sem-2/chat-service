GRPC_GEN_FILES=./proto/chats.proto

# Used in Dockerfile.dev for live reloading
start:
	./build/app --host 0.0.0.0 --port 80

# Generates all grpc stuff
generate:
	protoc --go_out=. --go_opt=paths=import --go-grpc_out=. --go-grpc_opt=paths=import $(GRPC_GEN_FILES)

build: generate
	go build -o ./bin/app cmd/main.go

run: build
	docker-compose up -d


all: generate run
