package utils

import (
	"bytes"
	"text/template"
)

func EvaluateTemplate(t *template.Template, data interface{}) string {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		log.Error().Msgf("Not able to parse template to string, %s", err.Error())
		return ""
	}

	return tpl.String()
}
