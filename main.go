package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"

	"gopkg.in/yaml.v1"
)

const ConfigName = ".tower.yml"

func main() {
	appMainFile := flag.String("m", "main.go", "path to your app's main file.")
	appPort := flag.String("p", "5000", "port of your app.")
	pxyPort := flag.String("r", "8080", "proxy port of your app.")
	verbose := flag.Bool("v", false, "show more stuff.")

	flag.Parse()

	args := flag.Args()
	if len(args) == 1 && args[0] == "init" {
		generateExampleConfig()
		return
	}

	startTower(*appMainFile, *appPort, *pxyPort, *verbose)
}

func generateExampleConfig() {
	_, file, _, _ := runtime.Caller(0)
	exampleConfig := path.Dir(file) + "/tower.yml"
	exec.Command("cp", exampleConfig, ConfigName).Run()
	fmt.Println("== Generated config file " + ConfigName)
}

var (
	app App
)

func startTower(appMainFile, appPort, pxyPort string, verbose bool) {
	watchedFiles := ""
	appBuildDir := ""
	contents, err := ioutil.ReadFile(ConfigName)
	if err != nil {
		fmt.Println(err)
	} else {
		newmap := map[string]string{}
		yamlErr := yaml.Unmarshal(contents, &newmap)
		if yamlErr != nil {
			fmt.Println(yamlErr)
		}
		appMainFile, _ = newmap["main"]
		appPort, _ = newmap["app_port"]
		pxyPort, _ = newmap["pxy_port"]
		appBuildDir, _ = newmap["app_buildDir"]
		watchedFiles, _ = newmap["watch"]
	}

	err = dialAddress("127.0.0.1:"+appPort, 1)
	if err == nil {
		fmt.Println("Error: port (" + appPort + ") already in used.")
		os.Exit(1)
	}

	if verbose {
		fmt.Println("== Application Info")
		fmt.Printf("  build app with: %s\n", appMainFile)
		fmt.Printf("  redirect requests from localhost:%s to localhost:%s\n\n", ProxyPort, appPort)
	}

	app = NewApp(appMainFile, appPort, appBuildDir)
	watcher := NewWatcher(app.Root, watchedFiles)
	watcher.OnChanged = func(file string) {
		app.BuildStart.Do(func() {
			app.FinishedBuild = false
			err := app.Build()
			if err != nil {
				fmt.Println(err)
			} else {
				app.FinishedBuild = true
			}
			app.BuildStart = &sync.Once{}
		})
	}
	proxy := NewProxy(&app, &watcher)
	proxy.Port = pxyPort
	go func() {
		mustSuccess(watcher.Watch())
	}()
	mustSuccess(proxy.Listen())
}
