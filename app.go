package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
)

const AppBin = "tmp/tower-server"

type App struct {
	Cmd             *exec.Cmd
	MainFile        string
	Port            string
	Name            string
	Root            string
	ListeningReturn bool
}

func NewApp(mainFile, port string) (app App) {
	app.MainFile = mainFile
	app.Port = port
	wd, _ := os.Getwd()
	app.Name = path.Base(wd)
	app.Root = path.Dir(mainFile)
	return
}

func (this *App) Start() (err error) {
	err = this.Build()
	if err != nil {
		return
	}

	fmt.Println("== Starting " + this.Name)
	this.Cmd = exec.Command(AppBin)
	this.Cmd.Stdout = os.Stdout
	this.Cmd.Stderr = os.Stderr

	err = this.Cmd.Start()
	if err != nil {
		return
	}

	err = waitForServer("127.0.0.1:" + this.Port)
	if err != nil {
		return errors.New("Fail to start " + this.Name)
	}

	this.RestartOnReturn()

	return
}

func (this *App) Restart() (err error) {
	this.Stop()
	return this.Start()
}

func (this *App) Stop() {
	if this.Cmd != nil {
		fmt.Println("== Stopping " + this.Name)
		this.Cmd.Process.Kill()
		this.Cmd = nil
	}
}

func (this *App) Build() (err error) {
	fmt.Println("== Building " + this.Name)
	out, _ := exec.Command("go", "build", "-o", AppBin, this.MainFile).CombinedOutput()
	if len(out) > 0 {
		return errors.New("Could not build app: " + string(out))
	}
	return nil
}

func (this *App) RestartOnReturn() {
	fmt.Println("   (Hit [return] to rebuild your app)")
	if this.ListeningReturn {
		return
	}

	this.ListeningReturn = true
	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			input, _ := in.ReadString('\n')
			if input == "\n" {
				this.Restart()
			}
		}
	}()
}
