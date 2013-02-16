package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
)

const (
	AppBin           = "tmp/tower-server"
	HttpPanicMessage = "http: panic serving"
)

type App struct {
	Cmd       *exec.Cmd
	MainFile  string
	Port      string
	Name      string
	Root      string
	KeyPress  bool
	LastError string
}

type StderrCapturer struct {
	app *App
}

func (this StderrCapturer) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), HttpPanicMessage) {
		app.LastError = string(p)
	}
	os.Stdout.Write([]byte("----------- Application Error -----------\n"))
	n, err = os.Stdout.Write(p)
	os.Stdout.Write([]byte("-----------------------------------------\n"))
	return
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
	this.Cmd.Stderr = StderrCapturer{this}

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
		msg := strings.Replace(string(out), "# command-line-arguments\n", "", 1)
		return errors.New(msg)
	}
	return nil
}

func (this *App) IsRunning() bool {
	return this.Cmd != nil
}

func (this *App) RestartOnReturn() {
	if this.KeyPress {
		return
	}
	this.KeyPress = true

	// Listen to keypress of "return" and restart the app automatically
	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			input, _ := in.ReadString('\n')
			if input == "\n" {
				this.Restart()
			}
		}
	}()

	// Listen to "^C" signal and stop the app properly
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig // wait for the "^C" signal
		fmt.Println("")
		this.Stop()
		os.Exit(0)
	}()
}
