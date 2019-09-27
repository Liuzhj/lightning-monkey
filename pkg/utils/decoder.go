package utils

import (
	"bytes"
	uuid "github.com/satori/go.uuid"
	"html/template"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func ObjectMetaFor(obj runtime.Object) (*v1.ObjectMeta, error) {
	v, err := conversion.EnforcePtr(obj)
	if err != nil {
		return nil, err
	}
	var meta *v1.ObjectMeta
	err = runtime.FieldPtr(v, "ObjectMeta", &meta)
	return meta, err
}

func DecodeYamlOrJson(content string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(content), nil, nil)
	if err != nil {
		return nil, err
	}

	meta, err := ObjectMetaFor(obj)
	if err != nil {
		return nil, err
	}
	o := meta.GetObjectMeta()
	if o.GetNamespace() == "" {
		o.SetNamespace("default")
	}
	return obj, err
}

func TemplateReplace(tp string, replacementSlots map[string]string) (string, error) {
	tmpl, err := template.New(uuid.NewV4().String()).Parse(tp)
	if err != nil {
		return "", err
	}
	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, replacementSlots)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
