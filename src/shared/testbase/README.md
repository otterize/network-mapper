# Testing instructions

In order to run the tests locally, “envtest” requires some k8s binaries. To install them, run once:

```shell
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
source <(setup-envtest use -p env)
sudo mkdir -p /usr/local/kubebuilder
sudo ln -s "$KUBEBUILDER_ASSETS" /usr/local/kubebuilder/bin
```
