package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"os"
	"os/exec"
	"path"
	"runtime"
)

func main() {
	appConfigFile := flag.String("c", "", "run \"tower init\" to get an example config.")
	appMainFile := flag.String("m", "main.go", "path to your app's main file.")
	appPort := flag.String("p", "5000", "port of your app.")
	verbose := flag.Bool("v", false, "show more stuff.")

	flag.Parse()

	args := flag.Args()
	if len(args) == 1 && args[0] == "init" {
		generateExampleConfig()
		return
	}

	startTower(*appConfigFile, *appMainFile, *appPort, *verbose)
}

func generateExampleConfig() {
	_, file, _, _ := runtime.Caller(0)
	exampleConfig := path.Dir(file) + "/tower.yml"
	exec.Command("mkdir", "-f", "configs").Run()
	exec.Command("cp", exampleConfig, "configs/tower.yml").Run()
	fmt.Println("Generated example config in configs/tower.yml")
}

var (
	app     App
	watcher Watcher
	proxy   Proxy
)

func startTower(configFile, appMainFile, appPort string, verbose bool) {
	if len(configFile) != 0 {
		config, err := yaml.ReadFile(configFile)
		if err != nil {
			fmt.Printf("Could not load config from %s\n", configFile)
			os.Exit(1)
		}
		appMainFile, _ = config.Get("main")
		appPort, _ = config.Get("port")
	}

	err := dialAddress("127.0.0.1:"+appPort, 1)
	if err == nil {
		fmt.Println("Error: port (" + appPort + ") already in used.")
		os.Exit(1)
	}

	if verbose {
		fmt.Println("== Application Info")
		fmt.Printf("  build app with: %s\n", appMainFile)
		fmt.Printf("  redirect requests from localhost:%s to localhost:%s\n\n", ProxyPort, appPort)
	}

	app = NewApp(appMainFile, appPort)
	watcher = NewWatcher(app.Root)
	proxy = NewProxy(&app, &watcher)

	go func() {
		mustSuccess(watcher.Watch())
	}()
	mustSuccess(proxy.Listen())
}

func stopTower() {
	app.Stop()
}
