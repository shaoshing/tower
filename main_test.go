package main

import (
	"fmt"
	"github.com/shaoshing/gotest"
	"io/ioutil"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

func TestCmd(t *testing.T) {
	assert.Test = t

	go startTower("", "", true)
	err := dialAddress("127.0.0.1:8000", 60)
	if err != nil {
		panic(err)
	}
	defer func() {
		app.Stop()
		fmt.Println("\n\n\n\n\n")
	}()

	assert.Equal("server 1", get("http://127.0.0.1:8000/"))
	assert.Equal("server 1", get("http://127.0.0.1:8000/?k=v1&k=v2&k1=v3")) // Test logging parameters
	assert.Equal("server 1", get("http://127.0.0.1:5000/"))

	app.Stop()
	concurrency := 10
	compileChan := make(chan bool)
	for i := 0; i < concurrency; i++ {
		go func() {
			get("http://127.0.0.1:8000/")
			compileChan <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-compileChan:
		case <-time.After(10 * time.Second):
			assert.TrueM(false, "Timeout on concurrency testing.")
		}
	}

	// test app exits unexpectedly
	assert.Contain("App quit unexpetedly", get("http://127.0.0.1:8000/exit")) // should restart the application

	// test error page
	highlightCode := `<strong>&nbsp;&nbsp;&nbsp;&nbsp;`
	assert.Contain("panic: Panic !!", get("http://127.0.0.1:8000/panic"))                   // should be able to detect panic
	assert.Contain(highlightCode+`panic(errors.New`, get("http://127.0.0.1:8000/panic"))    // should show code snippet
	assert.Contain(`<strong>36`, get("http://127.0.0.1:8000/panic"))                        // should show line number
	assert.Contain("runtime error: index out of range", get("http://127.0.0.1:8000/error")) // should be able to detect runtime error
	assert.Contain(highlightCode+`paths[0]`, get("http://127.0.0.1:8000/error"))            // should show code snippet
	assert.Contain(`<strong>17`, get("http://127.0.0.1:8000/error"))                        // should show line number

	defer exec.Command("git", "checkout", "test").Run()

	exec.Command("cp", "test/files/server2.go_", "test/server1.go").Run()
	time.Sleep(100 * time.Millisecond)
	assert.Equal("server 2", get("http://127.0.0.1:8000/"))

	exec.Command("cp", "test/files/error.go_", "test/server1.go").Run()
	assert.Match("Build Error", get("http://127.0.0.1:8000/"))
}

func get(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	b_body, _ := ioutil.ReadAll(resp.Body)
	return string(b_body)
}
