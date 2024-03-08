module github.com/otterize/network-mapper/src

go 1.21.5

toolchain go1.22.0

require (
	github.com/99designs/gqlgen v0.17.44
	github.com/Khan/genqlient v0.7.0
	github.com/amit7itz/goset v1.2.1
	github.com/bombsimon/logrusr/v3 v3.0.0
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.6.0
	github.com/google/gopacket v1.1.19
	github.com/google/uuid v1.6.0
	github.com/jellydator/ttlcache/v3 v3.1.1
	github.com/labstack/echo-contrib v0.15.0
	github.com/labstack/echo/v4 v4.11.3
	github.com/labstack/gommon v0.4.2
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de
	github.com/nxadm/tail v1.4.8
	github.com/oriser/regroup v0.0.0-20210730155327-fca8d7531263
	github.com/otterize/go-procnet v0.1.1
	github.com/otterize/intents-operator/src v0.0.0-20240308102113-306b434d2338
	github.com/otterize/nilable v0.0.0-20230421170720-4a38614e35a9
	github.com/prometheus/client_golang v1.18.0
	github.com/samber/lo v1.33.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.4
	github.com/vektah/gqlparser/v2 v2.5.11
	go.opentelemetry.io/otel v1.18.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.41.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.41.0
	go.opentelemetry.io/otel/metric v1.18.0
	go.opentelemetry.io/otel/sdk v1.18.0
	go.opentelemetry.io/otel/sdk/metric v0.41.0
	go.uber.org/mock v0.2.0
	golang.org/x/exp v0.0.0-20230124195608-d38c7dcee874
	golang.org/x/sync v0.6.0
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	sigs.k8s.io/controller-runtime v0.16.3
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/alexflint/go-arg v1.4.3 // indirect
	github.com/alexflint/go-scalar v1.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bugsnag/bugsnag-go/v2 v2.2.0 // indirect
	github.com/bugsnag/panicwrap v1.3.4 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sosodev/duration v1.2.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/urfave/cli/v2 v2.27.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xrash/smetrics v0.0.0-20231213231151-1d8dd44e695e // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.41.0 // indirect
	go.opentelemetry.io/otel/trace v1.18.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/mod v0.16.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/oauth2 v0.12.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.19.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230815205213-6bfd019c3878 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.28.3 // indirect
	k8s.io/component-base v0.28.3 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
