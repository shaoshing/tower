package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
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
		must(watchServerDir())
		must(startProxyServer())
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func buildServer() error {
	fmt.Println("Building Server")
	out, _ := exec.Command("go", "build", "-o", serverBin, mainFile).CombinedOutput()
	if len(out) > 0 {
		return errors.New("Could not build server: " + string(out))
	}
	return nil
}

var server *exec.Cmd

func startServer() error {
	if server != nil && !changed {
		return nil
	}

	if server != nil && changed {
		fmt.Println("Changed, stopping server")
		server.Process.Kill()
		server = nil
		changed = false
	}

	must(buildServer())
	fmt.Println("Starting Server")
	server = exec.Command(serverBin)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	return server.Start()
}

var proxy *httputil.ReverseProxy

func startProxyServer() error {
	fmt.Println("Starting Proxy Server")
	url, _ := url.ParseRequestURI("http://localhost:" + *serverPort)
	proxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", ServeRequest)
	err := http.ListenAndServe(port, nil)
	return err
}

func ServeRequest(w http.ResponseWriter, r *http.Request) {
	must(startServer())

	proxy.ServeHTTP(w, r)
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
