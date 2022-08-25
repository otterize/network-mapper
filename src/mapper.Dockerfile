FROM golang:1.18-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./mapper/...

FROM buildenv as test
# install dependencies for "envtest" package
RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && \
    source <(setup-envtest use -p env) && \
    mkdir -p /usr/local/kubebuilder && \
    ln -s "$KUBEBUILDER_ASSETS" /usr/local/kubebuilder/bin
RUN go test ./mapper/...

FROM test as builder
RUN go build -o /main ./mapper/cmd

FROM alpine as release
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /main /main
RUN chmod +x /main

EXPOSE 9090
ENTRYPOINT ["/main"]
