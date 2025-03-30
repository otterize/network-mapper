## Justfiles are modernized Makefiles.
## To install just:
##  - macOS: `brew install just`
##  - Ubuntu: `sudo apt install just`
##  - Fedora: `sudo dnf install just`
## To list tasks: `just --list` (or `just`)
## To run a task: `just <task>`
## To set a variable: `just <task> <variable>=<value>`
## e.g. `just build-images image-tag=latest`

image-tag := "local"

list-tasks:
    @just --list

generate:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    go generate ./...

build-mapper:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    go build -o ../bin/mapper ./mapper/cmd

build-kafka-watcher:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    go build -o ../bin/kafka-watcher ./kafka-watcher/cmd

build-sniffer:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    go build -o ../bin/sniffer ./sniffer/cmd

build: generate build-mapper build-kafka-watcher build-sniffer

build-mapper-image:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    docker buildx build \
        --platform linux/arm64 \
        -t otterize/network-mapper:{{image-tag}} \
        -f ../build/mapper.Dockerfile \
        .

build-kafka-watcher-image:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t otterize/kafka-watcher:{{image-tag}} \
        -f ../build/kafka-watcher.Dockerfile \
        .

build-sniffer-image:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd src/
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t otterize/sniffer:{{image-tag}} \
        -f ../build/sniffer.Dockerfile \
        .

build-images: generate build-mapper-image build-kafka-watcher-image build-sniffer-image
