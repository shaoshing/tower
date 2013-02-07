package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"
)

const (
	port      = ":8000"
	serverBin = "tmp/tower-server"
)

var serverPort = flag.String("port", "5000", "web service address")
var mainFile string

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Error: you must specify the main file.")
	} else {
		mainFile = args[0]
		startTower()
	}
}

func startTower() {
	must(watchServerDir())
	must(startProxyServer())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func buildServer() error {
	fmt.Println("== Building Server")
	out, _ := exec.Command("go", "build", "-o", serverBin, mainFile).CombinedOutput()
	if len(out) > 0 {
		return errors.New("Could not build server: " + string(out))
	}
	return nil
}

var server *exec.Cmd

func startServer() (err error) {
	if server != nil && !changed {
		return nil
	}

	if server != nil && changed {
		fmt.Println("== Changed, stopping server")
		stopServer()
		changed = false
	}

	err = buildServer()
	if err != nil {
		return err
	}

	fmt.Println("== Starting Server")
	server = exec.Command(serverBin)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr

	err = server.Start()
	if err != nil {
		return err
	}

	err = waitForServer("127.0.0.1:" + *serverPort)
	return err
}

func stopServer() {
	if server != nil {
		server.Process.Kill()
		server = nil
	}

}

func waitForServer(address string) error {
	for {
		select {
		case <-time.After(1 * time.Second):
			_, err := net.Dial("tcp", address)
			if err == nil {
				return nil
			}
		case <-time.After(1 * time.Minute):
			return errors.New("Fail to start server")
		}
	}
	return nil
}

var proxy *httputil.ReverseProxy

func startProxyServer() error {
	fmt.Println("== Listening to http://localhost:8000")
	url, _ := url.ParseRequestURI("http://localhost:" + *serverPort)
	proxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", serveRequest)
	err := http.ListenAndServe(port, nil)
	return err
}

func serveRequest(w http.ResponseWriter, r *http.Request) {
	logStartRequest(r)
	defer logEndRequest(w, r, time.Now())

	err := startServer()
	if err != nil {
		renderError(w, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy.ServeHTTP(w, r)
}

func renderError(w http.ResponseWriter, err error) {
	projectName := path.Base(path.Dir(mainFile))
	fmt.Fprintf(w, "Fail to build %s\n Errors: \n%s", projectName, err.Error())
}

var staticExp = regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|ico|swf|js|css|html)`)

func isStaticRequest(uri string) bool {
	return staticExp.Match([]byte(uri))
}

func logStartRequest(r *http.Request) {
	// TODO:
	// display params
	if !isStaticRequest(r.RequestURI) {
		fmt.Printf("\n\n\nStarted %s \"%s\" at %s\n", r.Method, r.RequestURI, time.Now().Format("2006-01-02 15:04:05 +700"))
	}
}

func logEndRequest(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	// TODO: display status code
	if !isStaticRequest(r.RequestURI) {
		fmt.Printf("Completed in %dms\n", time.Since(startTime)/time.Millisecond)
	}
}

var changed = false

func watchServerDir() error {
	dir := path.Dir(mainFile)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			file := <-watcher.Event
			log.Println("changed:", file.Name)
			changed = true
		}
	}()

	err = watcher.Watch(dir)
	if err != nil {
		return err
	}
	return nil
}
