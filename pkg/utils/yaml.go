package utils

import (
	pkgerr "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

var (
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

// DecodeYamlToUnstructured decodes YAML to Unstructured
func DecodeYamlToUnstructured(data []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode(data, nil, obj)
	if err != nil {
		return nil, pkgerr.Wrap(err, "Decode YAML to Unstructured failed. ")
	}

	return obj, nil
}
