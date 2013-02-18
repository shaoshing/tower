package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const ProxyPort = ":8000"

type Proxy struct {
	App          *App
	ReserveProxy *httputil.ReverseProxy
	Watcher      *Watcher
}

func NewProxy(app *App, watcher *Watcher) (proxy Proxy) {
	proxy.App = app
	proxy.Watcher = watcher
	return
}

func (this *Proxy) Listen() (err error) {
	fmt.Println("== Listening to http://localhost" + ProxyPort)
	url, _ := url.ParseRequestURI("http://localhost:" + this.App.Port)
	this.ReserveProxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		this.ServeRequest(w, r)
	})
	return http.ListenAndServe(ProxyPort, nil)
}

func (this *Proxy) ServeRequest(w http.ResponseWriter, r *http.Request) {
	mw := ResponseWriterWrapper{0, w}
	this.logStartRequest(r)
	defer this.logEndRequest(&mw, r, time.Now())

	if this.App.IsQuit() {
		fmt.Println("== App quit unexpetedly")
		this.App.Run()
	}

	if !this.App.IsRunning() || this.Watcher.Changed {
		err := this.App.Restart()
		if err != nil {
			RenderBuildError(&mw, this.App, err.Error())
			return
		}
		this.Watcher.Reset()
	}

	app.LastError = ""
	this.ReserveProxy.ServeHTTP(&mw, r)
	if len(app.LastError) != 0 {
		RenderAppError(&mw, this.App, app.LastError)
	}
}

var staticExp = regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|ico|swf|js|css|html|woff)`)

func (this *Proxy) isStaticRequest(uri string) bool {
	return staticExp.Match([]byte(uri))
}

func (this *Proxy) logStartRequest(r *http.Request) {
	if !this.isStaticRequest(r.RequestURI) {
		fmt.Printf("\n\n\nStarted %s \"%s\" at %s\n", r.Method, r.RequestURI, time.Now().Format("2006-01-02 15:04:05 +700"))
		params := this.formatRequestParams(r)
		if len(params) > 0 {
			fmt.Printf("  Parameters: %s\n", params)
		}
	}
}

func (this *Proxy) formatRequestParams(r *http.Request) string {
	r.ParseForm()
	if r.Form == nil {
		return ""
	}

	var params []string
	for key, vals := range r.Form {
		var strVals []string
		for _, val := range vals {
			strVals = append(strVals, `"`+val+`"`)
		}
		params = append(params, `"`+key+`":[`+strings.Join(strVals, ", ")+`]`)
	}
	return strings.Join(params, ", ")
}

func (this *Proxy) logEndRequest(mw *ResponseWriterWrapper, r *http.Request, startTime time.Time) {
	if !this.isStaticRequest(r.RequestURI) {
		fmt.Printf("Completed %d in %dms\n", mw.Status, time.Since(startTime)/time.Millisecond)
	}
}

// A response Wrapper to capture request's status code.
type ResponseWriterWrapper struct {
	Status int
	http.ResponseWriter
}

func (this *ResponseWriterWrapper) WriteHeader(status int) {
	this.Status = status
	this.ResponseWriter.WriteHeader(status)
}
