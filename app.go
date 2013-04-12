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
	"sync"
)

const (
	AppBin           = "/tmp/tower-app"
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

	start   *sync.Once
	restart *sync.Once
}

type StderrCapturer struct {
	app *App
}

func (this StderrCapturer) Write(p []byte) (n int, err error) {
	httpError := strings.Contains(string(p), HttpPanicMessage)

	if httpError {
		this.app.LastError = string(p)
		os.Stdout.Write([]byte("----------- Application Error -----------\n"))
		n, err = os.Stdout.Write(p)
		os.Stdout.Write([]byte("-----------------------------------------\n"))
	} else {
		n, err = os.Stdout.Write(p)
	}
	return
}

func NewApp(mainFile, port string) (app App) {
	app.MainFile = mainFile
	app.Port = port
	wd, _ := os.Getwd()
	app.Name = path.Base(wd)
	app.Root = path.Dir(mainFile)
	app.start = &sync.Once{}
	app.restart = &sync.Once{}
	return
}

func (this *App) Start(build bool) (err error) {
	this.start.Do(func() {
		if build {
			err = this.build()
			if err != nil {
				fmt.Println("== Fail to build " + this.Name)
				return
			}
		}

		err = this.run()
		if err != nil {
			err = errors.New("Fail to run " + this.Name)
			return
		}

		this.RestartOnReturn()
		this.start = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return
}

func (this *App) Restart() (err error) {
	this.restart.Do(func() {
		this.Stop()
		err = this.Start(true)
		this.restart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return
}

func (this *App) Stop() {
	if this.IsRunning() {
		fmt.Println("== Stopping " + this.Name)
		this.Cmd.Process.Kill()
		this.Cmd = nil
	}
}

func (this *App) run() (err error) {
	_, err = os.Stat(AppBin)
	if err != nil {
		return
	}

	fmt.Println("== Running " + this.Name)
	this.Cmd = exec.Command(AppBin)
	this.Cmd.Stdout = os.Stdout
	this.Cmd.Stderr = StderrCapturer{this}
	go func() {
		this.Cmd.Run()
	}()

	err = dialAddress("127.0.0.1:"+this.Port, 60)
	return
}

func (this *App) build() (err error) {
	fmt.Println("== Building " + this.Name)
	out, _ := exec.Command("go", "build", "-o", AppBin, this.MainFile).CombinedOutput()
	if len(out) > 0 {
		msg := strings.Replace(string(out), "# command-line-arguments\n", "", 1)
		fmt.Printf("----------- Build Error -----------\n%s-----------------------------------\n", msg)
		return errors.New(msg)
	}
	return nil
}

func (this *App) IsRunning() bool {
	return this.Cmd != nil && this.Cmd.ProcessState == nil
}

func (this *App) IsQuit() bool {
	return this.Cmd != nil && this.Cmd.ProcessState != nil
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
