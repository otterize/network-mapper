# Otterize Network Mapper

![Otter Manning Helm](./otterhelm.png)


![build](https://img.shields.io/static/v1?label=build&message=passing&color=success)
![go report](https://img.shields.io/static/v1?label=go%20report&message=A%2B&color=success)
[![GoDoc reference example](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/nanomsg.org/go/mangos/v2)
![openssf](https://img.shields.io/static/v1?label=openssf%20best%20practices&message=passing&color=success)
![community](https://img.shields.io/badge/slack-Otterize_Slack-orange.svg?logo=slack)

[About](#about) | [Quickstart](https://docs.otterize.com/documentation/quick-tutorials/network-mapper) | [How does the Network Mapper work?](#how-does-the-intents-operator-work) | [Docs](https://docs.otterize.com/documentation/k8s-operators/operator) | [Contributing](#contributing) | [Slack](#slack)

## About
The Otterize Network Mapper creates a map of in-cluster traffic by capturing DNS traffic and inspecting active connections, then resolving the IP addresses participating in connections to the Pods, and crawling up the ownership of the Pod until it reaches the root object. See [Service name resolution](#Service_name_resolution) to learn more.

You can then use the CLI to list the traffic by client, or export it as JSON or ClientIntents Kubernetes resources (YAML). ClientIntents can be consumed by the [Intents Operator](https://github.com/otterize/intents-operator) to apply network policies or Kafka ACLs to your cluster, and achieve zero trust.

Example output from the Quickstart guide:
```
checkoutservice calls:
  - orderservice

orderservice calls:
  - kafka
```

## How does the Network Mapper work?

### Components
- Sniffer - the sniffer is deployed to each node, and is responsible for capturing node-local DNS traffic and inspecting open connections.
- Mapper - the mapper is deployed once, and resolves service names using the Kubernetes API with traffic information reported by the sniffers.

### Service name resolution
Service name resolution is performed one of two ways:
1. If an `otterize/service-name` label is present, that name is used.
2. If not, a recursive look up is performed for the Kubernetes resource owner for a Pod until the root is reached. For example, if you have a `Deployment` named `client`, which then creates and owns a `ReplicaSet`, which then creates and owns a `Pod`, then the service name for that pod is `client` - same as the name of the `Deployment`.

The goal is to generate a mapping that speaks in the same language that dev teams use, whether or not a label has been set.

## Contributing
1. Feel free to fork and open a pull request! Include tests and document your code in [Godoc style](https://go.dev/blog/godoc)
2. In your pull request, please refer to an existing issue or open a new one.

## Slack
[Join the Otterize Slack!](https://join.slack.com/t/otterizeworkspace/shared_invite/zt-1fnbnl1lf-ub6wler4QrW6ZzIn2U9x1A)
