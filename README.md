# Otterize network mapper

<img title="Otter Manning Helm" src="./otterhelm.png" width=200 />


![build](https://github.com/otterize/network-mapper/actions/workflows/build.yaml/badge.svg)
![go report](https://img.shields.io/static/v1?label=go%20report&message=A%2B&color=success)
[![community](https://img.shields.io/badge/slack-Otterize_Slack-purple.svg?logo=slack)](https://joinslack.otterize.com)

[About](#about) | [Quick tutorial](https://docs.otterize.com/quick-tutorials/k8s-network-mapper) | [How does the network mapper work?](#how-does-the-intents-operator-work) | [Docs](https://docs.otterize.com/components/network-mapper/) | [Contributing](#contributing) | [Slack](#slack)

## About
The Otterize network mapper creates a map of in-cluster traffic by capturing DNS traffic and inspecting active connections, then resolving the IP addresses participating in connections to the pods, and crawling up the ownership of the pod until it reaches the root object. See [Service name resolution](#Service_name_resolution) to learn more.

You can then use the CLI to list the traffic by client, or export it as JSON or ClientIntents Kubernetes resources (YAML). ClientIntents can be consumed by the [intents operator](https://github.com/otterize/intents-operator) to apply network policies or Kafka ACLs to your cluster, and achieve zero trust.

Example output from the [quick tutorial](https://docs.otterize.com/quick-tutorials/k8s-network-mapper):
```
checkoutservice calls:
  - orderservice

orderservice calls:
  - kafka
```

## Installation instructions
### Network mapper helm chart
```bash
helm repo add otterize https://helm.otterize.com
helm repo update
helm install network-mapper otterize/network-mapper -n otterize-system --create-namespace --wait
```
### Otterize CLI
Mac
```bash
brew install otterize/otterize/otterize-cli
```
Linux 64-bit
```bash
wget https://get.otterize.com/otterize-cli/v0.1.5/otterize_Linux_x86_64.tar.gz
tar xf otterize_Linux_x86_64.tar.gz
sudo cp otterize /usr/local/bin
```
Windows
```bash
scoop bucket add otterize-cli https://github.com/otterize/scoop-otterize-cli
scoop update
scoop install otterize-cli
```
For all installation options check out the [guide](https://docs.otterize.com/k8s-installation/#install-the-otterize-cli).

## How does the network mapper work?

### Components
- Sniffer - the sniffer is deployed to each node, and is responsible for capturing node-local DNS traffic and inspecting open connections.
- Mapper - the mapper is deployed once, and resolves service names using the Kubernetes API with traffic information reported by the sniffers.

### Service name resolution
Service name resolution is performed one of two ways:
1. If an `otterize/service-name` label is present, that name is used.
2. If not, a recursive look up is performed for the Kubernetes resource owner for a pod until the root is reached. For example, if you have a `Deployment` named `client`, which then creates and owns a `ReplicaSet`, which then creates and owns a `Pod`, then the service name for that pod is `client` - same as the name of the `Deployment`.

The goal is to generate a mapping that speaks in the same language that dev teams use, whether or not a label has been set.

## Contributing
1. Feel free to fork and open a pull request! Include tests and document your code in [Godoc style](https://go.dev/blog/godoc)
2. In your pull request, please refer to an existing issue or open a new one.

## Slack
[Join the Otterize Slack!](https://joinslack.otterize.com)
