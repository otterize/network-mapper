package intentsprinter

import (
	"github.com/otterize/intents-operator/src/operator/api/v1alpha1"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync/atomic"
	"text/template"
)

type IntentsPrinter struct {
	printCount int64
}

const crdTemplate = `apiVersion: {{ .APIVersion }}
kind: {{ .Kind }}
metadata:
  name: {{ .Name }}
spec:
  service:
    name: {{ .Spec.Service.Name }}
  calls:
{{- range $intent := .Spec.Calls }}
    - name: {{ $intent.Name }}
      type: {{ $intent.Type }}
{{- if ne $intent.Namespace "" }}
      namespace: {{ $intent.Namespace }}
{{- end -}}
{{ end }}`

var crdTemplateParsed = template.Must(template.New("intents").Parse(crdTemplate))

// Keep this bit here so we have a compile time check that the structure the template assumes is correct.
var _ = v1alpha1.ClientIntents{
	TypeMeta:   v1.TypeMeta{Kind: "", APIVersion: ""},
	ObjectMeta: v1.ObjectMeta{Name: "", Namespace: ""},
	Spec: &v1alpha1.IntentsSpec{
		Service: v1alpha1.Service{Name: ""},
		Calls: []v1alpha1.Intent{{
			Type: "", Name: "", Namespace: "",
		}},
	},
}

func (p *IntentsPrinter) PrintObj(intents *v1alpha1.ClientIntents, w io.Writer) error {
	count := atomic.AddInt64(&p.printCount, 1)
	if count > 1 {
		if _, err := w.Write([]byte("\n---\n")); err != nil {
			return err
		}
	}
	return crdTemplateParsed.Execute(w, intents)
}
