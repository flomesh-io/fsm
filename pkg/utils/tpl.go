package utils

import (
	"bytes"
	"k8s.io/klog/v2"
	"text/template"
)

func EvaluateTemplate(t *template.Template, data interface{}) string {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		klog.Errorf("Not able to parse template to string, %s", err.Error())
		return ""
	}

	return tpl.String()
}
