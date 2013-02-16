package main

import (
	"github.com/shaoshing/gotest"
	"io/ioutil"
	"net/http"
	"os/exec"
	"testing"
)

func TestCmd(t *testing.T) {
	assert.Test = t

	go startTower("test/configs/tower.yml")
	err := waitForServer("127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	defer stopTower()

	assert.Equal("server 1", get("http://127.0.0.1:8000/"))
	assert.Equal("server 1", get("http://127.0.0.1:5000/"))

	// test error page
	highlightCode := `<strong>&nbsp;&nbsp;&nbsp;&nbsp;`
	assert.Contain("Panic !!", get("http://127.0.0.1:8000/panic"))                       // should be able to detect panic
	assert.Contain(highlightCode+`panic(errors.New`, get("http://127.0.0.1:8000/panic")) // should show code snippet
	assert.Contain(`<strong>30`, get("http://127.0.0.1:8000/panic"))                     // should show line number
	assert.Contain("index out of range", get("http://127.0.0.1:8000/error"))             // should be able to detect runtime error
	assert.Contain(highlightCode+`paths[0]`, get("http://127.0.0.1:8000/error"))         // should show code snippet
	assert.Contain(`<strong>16`, get("http://127.0.0.1:8000/error"))                     // should show line number

	defer exec.Command("git", "checkout", "test").Run()

	exec.Command("cp", "test/files/server2.go", "test/server1.go").Run()
	assert.Equal("server 2", get("http://127.0.0.1:8000/"))

	exec.Command("cp", "test/files/error.go", "test/server1.go").Run()
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
