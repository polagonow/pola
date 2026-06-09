package actionbridge

import "embed"

//go:embed _templates/*
var templates embed.FS

func mustReadTemplate(name string) string {
	b, err := templates.ReadFile("_templates/" + name)
	if err != nil {
		panic("codegen: missing template " + name + ": " + err.Error())
	}
	return string(b)
}

var (
	goTmpl = mustReadTemplate("go_bridge.tmpl")
	tsTmpl = mustReadTemplate("ts_generated.tmpl")
)
