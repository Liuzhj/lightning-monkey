package utils

import (
	"bytes"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"html/template"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	agg_v1betaObj "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
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
