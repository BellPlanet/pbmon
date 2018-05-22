package tpl

import "html/template"

var (
	Index *template.Template
)

func mustMakeTemplate(name string) *template.Template {
	p, err := Asset(name)
	if err != nil {
		panic(err)
	}

	return template.Must(template.New(name).Parse(string(p)))
}

func init() {
	Index = mustMakeTemplate("index.html")
}
