package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/kylelemons/go-gypsy/yaml"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

const (
	port = ":8000"
)

var appConfigFile = flag.String("config", "configs/tower.yml", "run \"tower init\" to get an example config.")
var app App

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 1 && args[0] == "init" {
		generateExampleConfig()
		return
	}

	startTower(*appConfigFile)
}

func generateExampleConfig() {
	_, file, _, _ := runtime.Caller(0)
	exampleConfig := path.Dir(file) + "/tower.yml"
	exec.Command("mkdir", "-f", "configs").Run()
	exec.Command("cp", exampleConfig, "configs/tower.yml").Run()
	fmt.Println("Generated example config in configs/tower.yml")
}

func startTower(configFile string) {
	config, err := yaml.ReadFile(configFile)
	if err != nil {
		fmt.Println("You must have a tower.yml config file, run \"tower init\" to get an example config.")
		return
	}

	appMainFile, _ := config.Get("main")
	appPort, _ := config.Get("port")
	app = NewApp(appMainFile, appPort)

	mustSuccess(app.Start())
	mustSuccess(watchServerDir())
	mustSuccess(startProxyServer())
}

func mustSuccess(err error) {
	if err != nil {
		panic(err)
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
			return errors.New("Fail to start " + app.Name)
		}
	}
	return nil
}

var proxy *httputil.ReverseProxy

func startProxyServer() error {
	fmt.Println("== Listening to http://localhost:8000")
	url, _ := url.ParseRequestURI("http://localhost:" + app.Port)
	proxy = httputil.NewSingleHostReverseProxy(url)

	http.HandleFunc("/", serveRequest)
	err := http.ListenAndServe(port, nil)
	return err
}

func serveRequest(w http.ResponseWriter, r *http.Request) {
	logStartRequest(r)
	defer logEndRequest(w, r, time.Now())

	if changed {
		err := app.Restart()
		if err != nil {
			renderError(w, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		changed = false
	}

	proxy.ServeHTTP(w, r)
}

func renderError(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "Fail to build %s\n Errors: \n%s", app.Name, err.Error())
}

var staticExp = regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|ico|swf|js|css|html|woff)`)

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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	expectedFileReg := regexp.MustCompile(`\.(go|html)`)
	go func() {
		for {
			file := <-watcher.Event
			if expectedFileReg.Match([]byte(file.Name)) {
				fmt.Println("== Change detected:", file.Name)
				changed = true
			}
		}
	}()

	ignoredPathReg := regexp.MustCompile(`(public)|(\/\.\w+)|(^\.)|(\.\w+$)`)
	dirsToWatch := make(map[string]bool)
	filepath.Walk(app.Root, func(filePath string, info os.FileInfo, e error) (err error) {
		if !info.IsDir() || ignoredPathReg.Match([]byte(filePath)) || dirsToWatch[filePath] {
			return
		}

		dirsToWatch[filePath] = true
		return
	})

	for dir, _ := range dirsToWatch {
		err = watcher.Watch(dir)
		if err != nil {
			return err
		}
	}
	return nil
}
