FROM golang:1.18-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
ARG GITHUB_TOKEN
RUN --mount=type=secret,id=github_token \
    if [ -f /run/secrets/github_token ]; then export GITHUB_TOKEN=$(cat /run/secrets/github_token); fi && \
    git config --global url."https://$GITHUB_TOKEN@github.com/".insteadOf "https://github.com/" &&  \
    go mod download &&  \
    git config --global --unset url."https://$GITHUB_TOKEN@github.com/".insteadOf

COPY . .
RUN go generate ./mapper/...

FROM buildenv as test
# install dependencies for "envtest" package
RUN ARCH=`arch`; if [ "$ARCH" == "x86_64" ]; then ARCH="amd64"; go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && \
    source <(setup-envtest use -p env --arch `arch`) && \
    mkdir -p /usr/local/kubebuilder && \
    ln -s "$KUBEBUILDER_ASSETS" /usr/local/kubebuilder/bin
RUN go test ./mapper/...

FROM test as builder
RUN CGO_ENABLED=0 go build -o /main ./mapper/cmd

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /main /main
USER 65532:65532

EXPOSE 9090
ENTRYPOINT ["/main"]
