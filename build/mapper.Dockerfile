FROM --platform=$BUILDPLATFORM golang:1.22.1-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM buildenv as test
# install dependencies for "envtest" package
RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20230216140739-c98506dc3b8e && \
    source <(setup-envtest use -p env) && \
    mkdir -p /usr/local/kubebuilder && \
    ln -s "$KUBEBUILDER_ASSETS" /usr/local/kubebuilder/bin
RUN go test ./mapper/...

FROM test as builder
ARG TARGETOS
ARG TARGETARCH

RUN echo "Building for $TARGETOS/$TARGETARCH - Running on $(uname -m)"
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /main ./mapper/cmd

# add version file
ARG VERSION
RUN echo -n $VERSION > /version

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /main /main
COPY --from=builder /version .
USER 65532:65532

EXPOSE 9090
ENTRYPOINT ["/main"]
