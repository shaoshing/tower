package main

import (
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var errorTemplate *template.Template

func init() {
	var err error
	templatePath := os.Getenv("GOPATH") + "/src/github.com/shaoshing/tower/error_page.html"
	errorTemplate, err = template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}
}

func RenderBuildError(w http.ResponseWriter, app *App, message string) {
	info := ErrorInfo{Title: "Fail to build " + app.Name, Message: template.HTML(message)}
	info.Prepare()

	renderPage(w, info)
}

func RenderAppError(w http.ResponseWriter, app *App, errMessage string) {
	info := ErrorInfo{Title: "Application Error:"}
	message, trace, _ := extractAppErrorInfo(errMessage)

	// from: 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Validation Error
	//   to: Validation Error
	message[0] = string(regexp.MustCompile(`.+\d+\.\d+.\d+.\d+\:\d+\:`).ReplaceAll([]byte(message[0]), []byte("")))
	info.Message = template.HTML(strings.Join(message, "\n"))

	for _, t := range trace {
		info.Trace = append(info.Trace, t[0], t[1])
	}
	info.ShowTrace = true

	info.Prepare()
	renderPage(w, info)
}

func renderPage(w http.ResponseWriter, info ErrorInfo) {
	err := errorTemplate.Execute(w, info)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func extractAppErrorInfo(errMessage string) (message []string, trace [][]string, appIndex int) {
	wd, _ := os.Getwd()

	items := make([][]string, 1)
	for iline, line := range strings.Split(errMessage, "\n") {
		if len(line) == 0 {
			continue
		}

		if line[0] == '/' {
			items = append(items, make([]string, 0))
		}
		if appIndex == 0 && strings.Index(line, wd) != -1 {
			appIndex = iline
		}

		i := len(items) - 1
		items[i] = append(items[i], line)
	}

	message = items[0]
	trace = items[1:]
	return
}

type ErrorInfo struct {
	Title   string
	Message template.HTML

	Trace     []string
	ShowTrace bool
}

func (this *ErrorInfo) Prepare() {
	this.TrimMessage()
}

func (this *ErrorInfo) TrimMessage() {
	html := strings.Join(strings.Split(string(this.Message), "\n"), "<br/>")
	this.Message = template.HTML(html)
}
