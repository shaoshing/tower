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
	fmt.Println("Building Server")
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
		fmt.Println("Changed, stopping server")
		stopServer()
		changed = false
	}

	err = buildServer()
	if err != nil {
		return err
	}

	fmt.Println("Starting Server")
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
	fmt.Println("Starting Proxy Server")
	url, _ := url.ParseRequestURI("http://localhost:" + *serverPort)
	proxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", serveRequest)
	err := http.ListenAndServe(port, nil)
	return err
}

func serveRequest(w http.ResponseWriter, r *http.Request) {
	err := startServer()
	if err != nil {
		renderError(w, err)
		return
	}

	proxy.ServeHTTP(w, r)
}

func renderError(w http.ResponseWriter, err error) {
	projectName := path.Base(path.Dir(mainFile))
	fmt.Fprintf(w, "Fail to build %s\n Errors: \n%s", projectName, err.Error())
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
