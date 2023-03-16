FROM golang:1.20.2

WORKDIR /app

COPY ./ /app

RUN go install github.com/githubnemo/CompileDaemon@latest

ENTRYPOINT CompileDaemon --build="make build" --command=./bin/edgeboxctl
