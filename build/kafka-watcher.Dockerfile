FROM --platform=linux/amd64 golang:1.19-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM buildenv as test
RUN go test ./exp/kafka-watcher/...

FROM test as builder
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /main ./exp/kafka-watcher/cmd

# add version file
ARG VERSION
RUN echo -n $VERSION > /version

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /main /main
COPY --from=builder /version .
USER 65532:65532

ENTRYPOINT ["/main"]
