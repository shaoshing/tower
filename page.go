package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var errorTemplate *template.Template

func init() {
	var err error
	templatePath := os.Getenv("GOPATH") + "/src/github.com/shaoshing/tower/page.html"
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

const SnippetLineNumbers = 13

func RenderAppError(w http.ResponseWriter, app *App, errMessage string) {
	info := ErrorInfo{Title: "Application Error:"}
	message, trace, appIndex := extractAppErrorInfo(errMessage)

	// from: 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Validation Error
	//   to: Validation Error
	message[0] = string(regexp.MustCompile(`.+\d+\.\d+.\d+.\d+\:\d+\:`).ReplaceAll([]byte(message[0]), []byte("")))
	info.Message = template.HTML(strings.Join(message, "\n"))

	for _, t := range trace {
		info.Trace = append(info.Trace, t[0], t[1])
	}
	info.ShowTrace = true

	// from: test/server1.go:16 (0x211e)
	//	 to: [test/server1.go, 16]
	appFileInfo := strings.Split(strings.Split(trace[appIndex][0], " ")[0], ":")
	// read the file
	content, err := ioutil.ReadFile(appFileInfo[0])
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")
	curLineNum, _ := strconv.ParseInt(appFileInfo[1], 10, 8)
	var code []Code
	for lineNum := curLineNum - SnippetLineNumbers/2 + 1; lineNum <= curLineNum+SnippetLineNumbers/2+1; lineNum++ {
		if int64(len(lines)) >= lineNum {
			code = append(code, Code{int(lineNum), template.HTML(lines[lineNum]), lineNum == curLineNum-1})
		}
	}
	info.ShowSnippet = true
	info.Snippet = code

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

// Example input
// 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Panic !!
// /usr/local/Cellar/go/1.0.3/src/pkg/net/http/server.go:589 (0x31ed9)
// _func_004: buf.Write(debug.Stack())
// /usr/local/Cellar/go/1.0.3/src/pkg/runtime/proc.c:1443 (0x10b83)
// panic: reflect·call(d->fn, d->args, d->siz);
// /Users/user/tower/test/server1.go:16 (0x211e)
// Panic: panic(errors.New("Panic !!"))

// Example output
// message:
//	[2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Panic !!]
// trace:
//  [
//	 [test/server1.go:16 (0x211e), Panic: panic(errors.New("Panic !!"))]
//	]
func extractAppErrorInfo(errMessage string) (message []string, trace [][]string, appIndex int) {
	// from: /Users/user/tower/test/server1.go:16 (0x211e)
	// 		   Panic: panic(errors.New("Panic !!"))
	//   to: <n>//Users/user/tower/test/server1.go:16 (0x211e)<n>Panic: panic(errors.New("Panic !!"))
	errMessage = strings.Replace(strings.Replace(errMessage, "\n", "<n>", -1), "<n>/", "<n>//", -1)

	wd, _ := os.Getwd()
	for i, line := range strings.Split(errMessage, "<n>/") {
		lines := strings.Split(line, "<n>")
		if i == 0 {
			message = lines
			continue
		}

		if appIndex == 0 && strings.Index(lines[0], wd) != -1 {
			appIndex = i - 1
		}
		trace = append(trace, lines)
	}
	return
}

type ErrorInfo struct {
	Title   string
	Message template.HTML

	Trace     []string
	ShowTrace bool

	Snippet     []Code
	ShowSnippet bool
}

type Code struct {
	Number  int
	Code    template.HTML
	Current bool
}

func (this *ErrorInfo) Prepare() {
	this.TrimMessage()
}

func (this *ErrorInfo) TrimMessage() {
	html := strings.Join(strings.Split(string(this.Message), "\n"), "<br/>")
	this.Message = template.HTML(html)
}
