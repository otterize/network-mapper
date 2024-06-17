package mapper

//go:generate go run github.com/99designs/gqlgen@v0.17.44
//go:generate ./fix-errors-import.sh
//go:generate go run go.uber.org/mock/mockgen@v0.2.0 -destination=./pkg/mocks/mock_k8s_client.go -package=mocks -mock_names Client=K8sClient,SubResourceWriter=K8sStatus sigs.k8s.io/controller-runtime/pkg/client Client,SubResourceWriter
//go:generate go run go.uber.org/mock/mockgen@v0.2.0 -destination=./pkg/mocks/mock_kubefinder.go -package=mocks -source=./pkg/resourcevisibility/svc_reconciler.go KubeFinder
