package utils

import (
	"bytes"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"html/template"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	agg_v1betaObj "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	agg_scheme "k8s.io/kube-aggregator/pkg/apiserver/scheme"
	"strings"
)

func ObjectMetaFor(obj interface{}) (*v1.ObjectMeta, error) {
	if newObj, isOK := obj.(runtime.Object); isOK {
		v, err := conversion.EnforcePtr(newObj)
		if err != nil {
			return nil, err
		}
		var meta *v1.ObjectMeta
		err = runtime.FieldPtr(v, "ObjectMeta", &meta)
		return meta, err
	}
	if newObj, isOK := obj.(*agg_v1betaObj.APIService); isOK {
		return &newObj.ObjectMeta, nil
	}
	return nil, fmt.Errorf("Unsupported object: %#v", obj)
}

func DecodeYamlOrJson(content string) (runtime.Object, error) {
	var decoder func(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error)
	if strings.Contains(content, "apiregistration.k8s.io") {
		decoder = agg_scheme.Codecs.UniversalDeserializer().Decode
	} else {
		decoder = scheme.Codecs.UniversalDeserializer().Decode
	}
	obj, _, err := decoder([]byte(content), nil, nil)
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
