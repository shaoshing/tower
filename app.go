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
	Cmds            map[string]*exec.Cmd
	MainFile        string
	Port            string
	Ports           []string
	BuildDir        string
	Name            string
	Root            string
	KeyPress        bool
	LastError       string
	PortParamName   string //端口参数名称(用于指定应用程序监听的端口，例如：webx.exe -p 8080，这里的-p就是端口参数名)
	SwitchToNewPort bool

	BuildStart   *sync.Once
	startErr     error
	AppRestart   *sync.Once
	restartErr   error
	portBinFiles map[string]string
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

func NewApp(mainFile, port, buildDir, portParamName string) (app App) {
	app.Cmds = make(map[string]*exec.Cmd)
	app.MainFile = mainFile
	app.BuildDir = buildDir
	app.PortParamName = portParamName
	app.ParseMutiPort(port)
	app.Port = app.UseRandPort()
	wd, _ := os.Getwd()
	app.Name = path.Base(wd)
	app.Root = path.Dir(mainFile)
	app.BuildStart = &sync.Once{}
	app.AppRestart = &sync.Once{}
	app.portBinFiles = make(map[string]string)
	return
}

func (this *App) ParseMutiPort(port string) {
	p := strings.Split(port, `,`)
	this.Ports = make([]string, 0)
	for _, v := range p {
		r := strings.Split(v, `-`)
		if len(r) > 1 {
			i, _ := strconv.Atoi(r[0])
			j, _ := strconv.Atoi(r[1])
			for ; i <= j; i++ {
				this.Ports = append(this.Ports, fmt.Sprintf("%v", i))
			}
		} else {
			this.Ports = append(this.Ports, r[0])
		}
	}
}

func (this *App) SupportMutiPort() bool {
	return this.Ports != nil && len(this.Ports) > 1 && this.PortParamName != ``
}

func (this *App) UseRandPort() string {
	for _, port := range this.Ports {
		if port != this.Port {
			return port
		}
	}
	return this.Port
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

		this.startErr = this.Run(this.Port)
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
		this.Stop(this.Port)
		this.restartErr = this.Start(true)
		this.AppRestart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return this.restartErr
}

func (this *App) BinFile(args ...string) (f string) {
	binFileName := AppBin
	if len(args) > 0 {
		binFileName = args[0]
	}
	if app.BuildDir != "" {
		f = filepath.Join(app.BuildDir, binFileName)
	} else {
		f = binFileName
	}
	if runtime.GOOS == "windows" {
		f += ".exe"
	}
	return
}

func (this *App) Stop(port string, args ...string) {
	if this.IsRunning(port) {
		fmt.Println("== Stopping " + this.Name)
		cmd := this.GetCmd(port)
		cmd.Process.Kill()
		cmd = nil
		os.Remove(this.BinFile(args...))
		delete(this.Cmds, port)
		delete(this.portBinFiles, port)
	}
}

func (this *App) Clean() {
	for port, cmd := range this.Cmds {
		if port == this.Port || !CmdIsRunning(cmd) {
			continue
		}
		fmt.Println("== Stopping app at port: " + port)
		cmd.Process.Kill()
		cmd = nil
		if bin, ok := this.portBinFiles[port]; ok && bin != "" {
			err := os.Remove(bin)
			if err != nil {
				fmt.Sprintln(err)
			}
		}
		delete(this.Cmds, port)
		delete(this.portBinFiles, port)
	}
}

func (this *App) GetCmd(args ...string) (cmd *exec.Cmd) {
	var port string
	if len(args) > 0 {
		port = args[0]
	} else {
		port = this.Port
	}
	cmd, _ = this.Cmds[port]
	return
}

func (this *App) SetCmd(port string, cmd *exec.Cmd) {
	this.Cmds[port] = cmd
}

func (this *App) Run(port string) (err error) {
	bin := this.BinFile()
	_, err = os.Stat(bin)
	if err != nil {
		return
	}
	fmt.Println("== Running at port " + port + ": " + this.Name)
	if this.Port != port {
		this.SwitchToNewPort = true
	}
	this.Port = port //记录被使用的端口，避免下次使用
	var cmd *exec.Cmd
	/*
		cmd = this.GetCmd()
		if cmd != nil {
			this.Stop(port)
		}
	*/
	this.portBinFiles[port] = bin
	if this.SupportMutiPort() {
		cmd = exec.Command(bin, this.PortParamName, port)
	} else {
		cmd = exec.Command(bin)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = StderrCapturer{this}
	go func() {
		cmd.Run()
	}()
	this.SetCmd(this.Port, cmd)
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
	fmt.Println("== Build completed.")
	return nil
}

func (this *App) IsRunning(args ...string) bool {
	return CmdIsRunning(this.GetCmd(args...))
}

func CmdIsRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState == nil
}

func CmdIsQuit(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState != nil
}

func (this *App) IsQuit(args ...string) bool {
	return CmdIsQuit(this.GetCmd(args...))
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
		this.Stop(this.Port)
		os.Exit(0)
	}()
}
