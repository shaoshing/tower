package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
)

const (
	port      = ":8000"
	serverBin = "tmp/tower-server"
)

var serverPort = flag.String("port", "5000", "web service address")

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Error: you must specify the main file.")
	} else {
		mainFile := args[0]
		must(buildServer(mainFile))
		must(startServer())
		must(startProxyServer())
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func buildServer(mainFile string) error {
	fmt.Println("Building Server")
	out, _ := exec.Command("go", "build", "-o", serverBin, mainFile).CombinedOutput()
	if len(out) > 0 {
		return errors.New("Could not build server: " + string(out))
	}
	return nil
}

func startServer() error {
	fmt.Println("Starting Server")
	cmd := exec.Command(serverBin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
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
	proxy.ServeHTTP(w, r)
}
