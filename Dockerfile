# syntax=docker/dockerfile:1.3.1
FROM golang:1.20 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.* .

# Set environment variables
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPRIVATE=github.com/praetorian-inc

RUN go mod download

# Copy the go source
COPY . .
#RUN CGO_ENABLED=0 go build -trimpath -o konstellation ./cmd/cmd
ENTRYPOINT ["/workspace/konstellation"]
#FROM golang:1.20
#COPY --from=builder /workspace/konstellation /home/
#WORKDIR /home
#USER root
#ENTRYPOINT ["/home/konstellation"]
