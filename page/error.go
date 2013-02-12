package page

import (
	"html/template"
	"net/http"
	"os"
	"strings"
)

type ErrorInfo struct {
	Title   string
	Message string

	MessageHtml template.HTML
}

func (this *ErrorInfo) Prepare() {
	this.TrimMessage()
}

func (this *ErrorInfo) TrimMessage() {
	html := strings.Join(strings.Split(this.Message, "\n"), "<br/>")
	this.MessageHtml = template.HTML(html)
}

var errorTemplate *template.Template

func init() {
	var err error
	templatePath := os.Getenv("GOPATH") + "/src/github.com/shaoshing/tower/page/error.html"
	errorTemplate, err = template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}
}

func RenderError(w http.ResponseWriter, appErr ErrorInfo) {
	appErr.Prepare()
	err := errorTemplate.Execute(w, appErr)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusInternalServerError)
}
