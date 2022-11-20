# Otterize network mapper

<img title="Otter Manning Helm" src="./otterhelm.png" width=200 />


![build](https://github.com/otterize/network-mapper/actions/workflows/build.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/otterize/network-mapper/src)](https://goreportcard.com/report/github.com/otterize/network-mapper/src)
[![community](https://img.shields.io/badge/slack-Otterize_Slack-purple.svg?logo=slack)](https://joinslack.otterize.com)

* [About](#about)
* [Try the network mapper](#try-the-network-mapper)
* [Installation instructions](#installation-instructions)
* [How does the network mapper work?](#how-does-the-network-mapper-work)
  * [Components](#components)
  * [Service name resolution](#service-name-resolution)
  * [Why should I use this over a service mesh?](#difference-from-a-service-mesh)
* [Using the network map](#using-the-network-map)
* [Learn more](#learn-more)
* [Contributing](#contributing)
* [Slack](#slack)

https://user-images.githubusercontent.com/29180932/202178239-e93f305b-33aa-4caf-88f7-a78d666e071a.mp4

## About
The Otterize network mapper creates a map of in-cluster traffic by capturing DNS traffic and inspecting active connections then resolving the IP addresses participating in connections to the pods, and crawling up the ownership of the pod until it reaches the root object. See [service name resolution](#service-name-resolution) to learn more. The network mapper continues building the network map as long as it's deployed.

You can use the [Otterize CLI](https://github.com/otterize/otterize-cli) to list the traffic by client, reset the traffic the mapper remembers, or export it as JSON or YAML.

Example output from running the network mapper on the [Google Cloud microservices demo](https://github.com/GoogleCloudPlatform/microservices-demo):
```bash
$ otterize mapper list
cartservice in namespace otterize-ecom-demo calls:
  - redis-cart
checkoutservice in namespace otterize-ecom-demo calls:
  - cartservice
  - currencyservice
  - emailservice
  - paymentservice
  - productcatalogservice
  - shippingservice
frontend in namespace otterize-ecom-demo calls:
  - adservice
  - cartservice
  - checkoutservice
  - currencyservice
  - productcatalogservice
  - recommendationservice
  - shippingservice
loadgenerator in namespace otterize-ecom-demo calls:
  - frontend
recommendationservice in namespace otterize-ecom-demo calls:
  - productcatalogservice
```
## Try the network mapper
Try the [quick tutorial guide](https://docs.otterize.com/quick-tutorials/k8s-network-mapper) to get a hands-on experience in 5 minutes.

## Installation instructions
### Install and run the network mapper using Helm
```bash
helm repo add otterize https://helm.otterize.com
helm repo update
helm install network-mapper otterize/network-mapper -n otterize-system --create-namespace --wait
```
### Install Otterize CLI to query data from the network mapper
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
For more platforms, see [the installation guide](https://docs.otterize.com/k8s-installation/#install-the-otterize-cli).

## How does the network mapper work?

### Components
- Sniffer: the sniffer is deployed to each node, and is responsible for capturing node-local DNS traffic and inspecting open connections.
- Mapper: the mapper is deployed once, and resolves service names using the Kubernetes API with traffic information reported by the sniffers.

### Service name resolution
Service names are resolved in one of two ways:
1. If an `otterize/service-name` label is present, that name is used.
2. If not, a recursive look-up is performed for the Kubernetes resource owner for a pod until the root is reached.
For example, if you have a `Deployment` named `client`, which then creates and owns a `ReplicaSet`, 
which then creates and owns a `Pod`, then the service name for that pod is `client` - same as the name of the `Deployment`.
The goal is to generate a mapping that speaks in the same language that dev teams use.

### Difference from a service mesh
the mapper is a zero-config tool that aims to be lightweight and doesn't require you to adapt anything in your cluster. It 
does not try to replace service mesh in any way, but rather give you insights about traffic in your cluster without 
a complete overhaul or the need to adapt to it.

## Using the network map
The network mapper continuously builds a map of pod to pod communication in the cluster. The map can be exported at any time in either JSON or YAML formats with the Otterize CLI.

The YAML export is formatted as `ClientIntents` Kubernetes resource files. Client intents files can be consumed by the [Otterize intents operator](https://github.com/otterize/intents-operator) to configure pod-to-pod access with network policies, or Kafka client access with Kafka ACLs and mTLS.

## Learn more
Explore our [documentation](https://docs.otterize.com/) site to learn how to:
- [Map pod-to-pod communication](https://docs.otterize.com/guides/k8s-mapping-pod-to-pod-calls).
- [Automate network policies](https://docs.otterize.com/quick-tutorials/k8s-network-policies).
- And more...

## Contributing
1. Feel free to fork and open a pull request! Include tests and document your code in [Godoc style](https://go.dev/blog/godoc)
2. In your pull request, please refer to an existing issue or open a new one.
3. See our [Contributor License Agreement](https://github.com/otterize/cla/).

## Slack
To join the conversation, ask questions, and engage with other users, join the Otterize Slack!

[![button](https://i.ibb.co/vwRP6xK/Group-3090-2.png)](https://joinslack.otterize.com)
