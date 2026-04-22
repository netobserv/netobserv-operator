ARG TARGETARCH

# Build the manager binary
FROM docker.io/library/golang:1.25 as builder

ARG TARGETARCH=amd64
ARG LDFLAGS

WORKDIR /opt/app-root

# Copy the go manifests and source
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/
COPY main.go main.go
COPY api/ api/
COPY internal/ internal/
COPY config/crd/bases config/crd/bases

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO111MODULE=on go build -ldflags "$LDFLAGS" -mod vendor -a -o manager main.go

# Create final image from minimal + built binary
FROM --platform=linux/$TARGETARCH registry.access.redhat.com/ubi9/ubi-minimal:9.7-1776833838
WORKDIR /
COPY --from=builder /opt/app-root/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
