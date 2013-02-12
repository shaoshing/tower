package page

import (
	"html/template"
	"net/http"
	"os"
)

type Error struct {
	Title   string
	Message string
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

func RenderError(w http.ResponseWriter, appErr Error) {
	err := errorTemplate.Execute(w, appErr)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusInternalServerError)
}
