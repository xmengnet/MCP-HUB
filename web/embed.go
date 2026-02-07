// Package web 嵌入网页模板资源
package web

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Templates 解析所有模板
var Templates *template.Template

func init() {
	Templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))
}
