FROM golang:1.19-alpine3.17 as builder

WORKDIR /app

COPY . .

RUN apk update \
    && apk add make protobuf-dev \
    && go mod download \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 \
    && export PATH="$PATH:$(go env GOPATH)/bin" \
    && make build

FROM alpine:3.17

COPY --from=builder /app/bin/app /server

CMD /server --host 0.0.0.0 --port 80
