package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
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
	this.App.Start()

	fmt.Println("== Listening to http://localhost" + ProxyPort)
	url, _ := url.ParseRequestURI("http://localhost:" + this.App.Port)
	this.ReserveProxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		this.ServeRequest(w, r)
	})
	return http.ListenAndServe(ProxyPort, nil)
}

func (this *Proxy) ServeRequest(w http.ResponseWriter, r *http.Request) {
	this.logStartRequest(r)
	defer this.logEndRequest(w, r, time.Now())

	if this.Watcher.Changed {
		err := this.App.Restart()
		if err != nil {
			this.renderError(w, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		this.Watcher.Reset()
	}

	this.ReserveProxy.ServeHTTP(w, r)
}

func (this *Proxy) renderError(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "Fail to build %s\n Errors: \n%s", this.App.Name, err.Error())
}

var staticExp = regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|ico|swf|js|css|html|woff)`)

func (this *Proxy) isStaticRequest(uri string) bool {
	return staticExp.Match([]byte(uri))
}

func (this *Proxy) logStartRequest(r *http.Request) {
	// TODO:
	// display params
	if !this.isStaticRequest(r.RequestURI) {
		fmt.Printf("\n\n\nStarted %s \"%s\" at %s\n", r.Method, r.RequestURI, time.Now().Format("2006-01-02 15:04:05 +700"))
	}
}

func (this *Proxy) logEndRequest(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	// TODO: display status code
	if !this.isStaticRequest(r.RequestURI) {
		fmt.Printf("Completed in %dms\n", time.Since(startTime)/time.Millisecond)
	}
}
