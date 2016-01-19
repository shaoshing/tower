package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	HttpPanicMessage = "http: panic serving"
)

var (
	AppBin = "tower-app-" + strconv.FormatInt(time.Now().Unix(), 10)
)

type App struct {
	Cmd           *exec.Cmd
	MainFile      string
	Port          string
	BuildDir      string
	Name          string
	Root          string
	KeyPress      bool
	LastError     string
	FinishedBuild bool

	BuildStart *sync.Once
	startErr   error
	AppRestart *sync.Once
	restartErr error
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

func NewApp(mainFile, port, buildDir string) (app App) {
	app.MainFile = mainFile
	app.Port = port
	app.BuildDir = buildDir
	wd, _ := os.Getwd()
	app.Name = path.Base(wd)
	app.Root = path.Dir(mainFile)
	app.BuildStart = &sync.Once{}
	app.AppRestart = &sync.Once{}
	return
}

func (this *App) Start(build bool) error {
	this.BuildStart.Do(func() {
		if build {
			this.startErr = this.Build()
			if this.startErr != nil {
				fmt.Println("== Fail to build " + this.Name)
				this.BuildStart = &sync.Once{}
				return
			}
		}

		this.startErr = this.Run()
		if this.startErr != nil {
			this.startErr = errors.New("Fail to run " + this.Name)
			this.BuildStart = &sync.Once{}
			return
		}

		this.RestartOnReturn()
		this.BuildStart = &sync.Once{}
	})

	return this.startErr
}

func (this *App) Restart() error {
	this.AppRestart.Do(func() {
		this.Stop()
		this.restartErr = this.Start(this.FinishedBuild == false)
		this.AppRestart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return this.restartErr
}

func (this *App) BinFile() (f string) {
	if app.BuildDir != "" {
		f = filepath.Join(app.BuildDir, AppBin)
	} else {
		f = AppBin
	}
	if runtime.GOOS == "windows" {
		f += ".exe"
	}
	return
}

func (this *App) Stop() {
	if this.IsRunning() {
		if this.FinishedBuild == false {
			os.Remove(this.BinFile())
		}
		fmt.Println("== Stopping " + this.Name)
		this.Cmd.Process.Kill()
		this.Cmd = nil
	}
}

func (this *App) Run() (err error) {
	_, err = os.Stat(this.BinFile())
	if err != nil {
		return
	}

	fmt.Println("== Running " + this.Name)
	this.Cmd = exec.Command(this.BinFile())
	this.Cmd.Stdout = os.Stdout
	this.Cmd.Stderr = StderrCapturer{this}
	go func() {
		this.Cmd.Run()
	}()
	this.FinishedBuild = false
	err = dialAddress("127.0.0.1:"+this.Port, 60)
	return
}

func (this *App) Build() (err error) {
	fmt.Println("== Building " + this.Name)
	AppBin = "tower-app-" + strconv.FormatInt(time.Now().Unix(), 10)
	out, _ := exec.Command("go", "build", "-o", this.BinFile(), this.MainFile).CombinedOutput()
	if len(out) > 0 {
		msg := strings.Replace(string(out), "# command-line-arguments\n", "", 1)
		fmt.Printf("----------- Build Error -----------\n%s-----------------------------------\n", msg)
		return errors.New(msg)
	}
	fmt.Println("== Build completed")
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
