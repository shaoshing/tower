package main

import (
	"html"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var errorTemplate *template.Template

func init() {
	_, filename, _, _ := runtime.Caller(1)
	pkgPath := path.Dir(filename)
	templatePath := pkgPath + "/page.html"

	var err error
	errorTemplate, err = template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}
}

func RenderError(w http.ResponseWriter, app *App, message string) {
	info := ErrorInfo{Title: "Error", Message: template.HTML(message)}
	info.Prepare()

	renderPage(w, info)
}

func RenderBuildError(w http.ResponseWriter, app *App, message string) {
	info := ErrorInfo{Title: "Build Error", Message: template.HTML(message)}
	info.Prepare()

	renderPage(w, info)
}

const SnippetLineNumbers = 13

func RenderAppError(w http.ResponseWriter, app *App, errMessage string) {
	info := ErrorInfo{Title: "Application Error"}
	message, trace, appIndex := extractAppErrorInfo(errMessage)

	// from: 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Validation Error
	//   to: Validation Error
	message[0] = string(regexp.MustCompile(`.+\d+\.\d+.\d+.\d+\:\d+\: `).ReplaceAll([]byte(message[0]), []byte("")))
	if !strings.Contains(message[0], "runtime error") {
		message[0] = "panic: " + message[0]
	}

	info.Message = template.HTML(strings.Join(message, "\n"))
	info.Trace = trace
	info.ShowTrace = true

	// from: test/server1.go:16 (0x211e)
	//	 to: [test/server1.go, 16]
	appFileInfo := strings.Split(strings.Split(trace[appIndex].File, " ")[0], ":")
	info.SnippetPath = appFileInfo[0]
	info.ShowSnippet = true
	curLineNum, _ := strconv.ParseInt(appFileInfo[1], 10, 16)
	info.Snippet = extractAppSnippet(appFileInfo[0], int(curLineNum))

	info.Prepare()
	renderPage(w, info)
}

func renderPage(w http.ResponseWriter, info ErrorInfo) {
	err := errorTemplate.Execute(w, info)
	if err != nil {
		panic(err)
	}
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
func extractAppErrorInfo(errMessage string) (message []string, trace []Trace, appIndex int) {
	// from: /Users/user/tower/test/server1.go:16 (0x211e)
	// 		   Panic: panic(errors.New("Panic !!"))
	//   to: <n>//Users/user/tower/test/server1.go:16 (0x211e)<n>Panic: panic(errors.New("Panic !!"))
	errMessage = strings.Replace(strings.Replace(errMessage, "\n", "<n>", -1), "<n>/", "<n>//", -1)

	wd, _ := os.Getwd()
	wd = wd + "/"
	for i, line := range strings.Split(errMessage, "<n>/") {
		lines := strings.Split(line, "<n>")
		if i == 0 {
			message = lines
			continue
		}

		t := Trace{Func: lines[1]}
		if strings.Index(lines[0], wd) != -1 {
			if appIndex == 0 {
				appIndex = i - 1
			}
			t.AppFile = true
		}
		t.File = strings.Replace(lines[0], wd, "", 1)
		// from: /Users/user/tower/test/server1.go:16 (0x211e)
		//   to: /Users/user/tower/test/server1.go:16
		t.File = string(regexp.MustCompile(`\(.+\)$`).ReplaceAll([]byte(t.File), []byte("")))
		trace = append(trace, t)
	}
	return
}

func extractAppSnippet(appFile string, curLineNum int) (snippet []Snippet) {
	content, err := ioutil.ReadFile(appFile)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")
	for lineNum := curLineNum - SnippetLineNumbers/2; lineNum <= curLineNum+SnippetLineNumbers/2; lineNum++ {
		if len(lines) >= lineNum {
			c := html.EscapeString(lines[lineNum-1])
			c = strings.Replace(c, "\t", "&nbsp;&nbsp;&nbsp;&nbsp;", -1)
			c = strings.Replace(c, " ", "&nbsp;", -1)
			snippet = append(snippet, Snippet{lineNum, template.HTML(c), lineNum == curLineNum})
		}
	}
	return
}

type ErrorInfo struct {
	Title   string
	Time    string
	Message template.HTML

	Trace     []Trace
	ShowTrace bool

	SnippetPath string
	Snippet     []Snippet
	ShowSnippet bool
}

type Snippet struct {
	Number  int
	Code    template.HTML
	Current bool
}

type Trace struct {
	File    string
	Func    string
	AppFile bool
}

func (this *ErrorInfo) Prepare() {
	this.TrimMessage()
	this.Time = time.Now().Format("15:04:05")
}

func (this *ErrorInfo) TrimMessage() {
	html := strings.Join(strings.Split(string(this.Message), "\n"), "<br/>")
	this.Message = template.HTML(html)
}
