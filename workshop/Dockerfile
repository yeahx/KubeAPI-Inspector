# Build the manager binary
FROM golang:1.23 as builder

WORKDIR /workspace
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

ENV GO111MODULE on
ENV DEBUG true
ENV GOPROXY http://goproxy.cn,direct

# Build workshop-apiserver
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -o workshop-apiserver cmd/main.go

FROM gcr.io/distroless/static-debian12:debug
WORKDIR /
COPY --from=builder /workspace/workshop-apiserver .

ENTRYPOINT ["/workshop-apiserver"]