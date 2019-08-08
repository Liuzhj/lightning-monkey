package utils

import (
	"bytes"
	uuid "github.com/satori/go.uuid"
	"html/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/pkg/api"
)

func DecodeYamlOrJson(content string) (runtime.Object, error) {
	decode := api.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(content), nil, nil)
	if err != nil {
		return nil, err
	}
	meta, err := metav1.ObjectMetaFor(obj)
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
