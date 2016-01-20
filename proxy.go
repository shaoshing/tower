package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const ProxyPort = "8080"

type Proxy struct {
	App          *App
	ReserveProxy *httputil.ReverseProxy
	Watcher      *Watcher
	FirstRequest *sync.Once
	Port         string
}

func NewProxy(app *App, watcher *Watcher) (proxy Proxy) {
	proxy.App = app
	proxy.Watcher = watcher
	proxy.Port = ProxyPort
	return
}

func (this *Proxy) Listen() (err error) {
	fmt.Println("== Listening to http://localhost:" + this.Port)
	url, _ := url.ParseRequestURI("http://localhost:" + this.App.Port)
	this.ReserveProxy = httputil.NewSingleHostReverseProxy(url)
	this.FirstRequest = &sync.Once{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		this.ServeRequest(w, r)
	})
	return http.ListenAndServe(":"+this.Port, nil)
}

func (this *Proxy) ServeRequest(w http.ResponseWriter, r *http.Request) {
	mw := ResponseWriterWrapper{ResponseWriter: w}
	this.logStartRequest(r)
	defer this.logEndRequest(&mw, r, time.Now())

	if this.App.SwitchToNewPort {
		this.App.SwitchToNewPort = false
		url, _ := url.ParseRequestURI("http://localhost:" + this.App.Port)
		this.ReserveProxy = httputil.NewSingleHostReverseProxy(url)
		this.FirstRequest.Do(func() {
			this.ReserveProxy.ServeHTTP(&mw, r)
			this.FirstRequest = &sync.Once{}
		})
		this.App.Clean()
	} else if !this.App.IsRunning() || this.Watcher.Changed {
		this.Watcher.Reset()
		err := this.App.Restart()
		if err != nil {
			RenderBuildError(&mw, this.App, err.Error())
			return
		}

		this.FirstRequest.Do(func() {
			this.ReserveProxy.ServeHTTP(&mw, r)
			this.FirstRequest = &sync.Once{}
		})
	}

	this.App.LastError = ""

	if !mw.Processed {
		this.ReserveProxy.ServeHTTP(&mw, r)
	}

	if len(this.App.LastError) != 0 {
		RenderAppError(&mw, this.App, this.App.LastError)
	}

	if this.App.IsQuit() {
		fmt.Println("== App quit unexpetedly")
		this.App.Start(false)
		RenderError(&mw, this.App, "App quit unexpetedly.")
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

type MyReadCloser struct {
	bytes.Buffer
}

func (this *MyReadCloser) Close() error {
	return nil
}

func (this *Proxy) formatRequestParams(r *http.Request) string {
	// Keep an copy of request Body, and restore it after parsed form.
	var b1, b2 MyReadCloser
	io.Copy(&b1, r.Body)
	io.Copy(&b2, &b1)
	r.Body = &b1
	r.ParseForm()
	r.Body = &b2

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
	Status    int
	Processed bool
	http.ResponseWriter
}

func (this *ResponseWriterWrapper) WriteHeader(status int) {
	this.Status = status
	this.Processed = true
	this.ResponseWriter.WriteHeader(status)
}
