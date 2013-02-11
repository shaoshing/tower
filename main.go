package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"os/exec"
	"path"
	"runtime"
)

var appConfigFile = flag.String("config", "configs/tower.yml", "run \"tower init\" to get an example config.")

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

var (
	app     App
	watcher Watcher
	proxy   Proxy
)

func startTower(configFile string) {
	config, err := yaml.ReadFile(configFile)
	if err != nil {
		fmt.Println("You must have a tower.yml config file, run \"tower init\" to get an example config.")
		return
	}

	appMainFile, _ := config.Get("main")
	appPort, _ := config.Get("port")
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
