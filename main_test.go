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
	defer stopServer()

	assert.Equal("server 1", get("http://127.0.0.1:8000/"))
	assert.Equal("server 1", get("http://127.0.0.1:5000/"))

	defer exec.Command("git", "checkout", "test").Run()

	exec.Command("cp", "test/files/server2.go", "test/server1.go").Run()
	assert.Equal("server 2", get("http://127.0.0.1:8000/"))

	exec.Command("cp", "test/files/error.go", "test/server1.go").Run()
	assert.Match("Fail to build test", get("http://127.0.0.1:8000/"))
}

func get(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	b_body, _ := ioutil.ReadAll(resp.Body)
	return string(b_body)
}
