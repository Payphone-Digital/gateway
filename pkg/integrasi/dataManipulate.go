package integrasi

import (
	"bytes"
	"html/template"

	"github.com/Masterminds/sprig/v3"
)

func RenderTemplateWithSprig(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("dynamic").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}
