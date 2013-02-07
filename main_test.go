package main

import (
	"github.com/bmizerany/assert"
	"io/ioutil"
	"net/http"
	"os/exec"
	"testing"
)

func TestCmd(t *testing.T) {
	mainFile = "test/server1.go"
	go func() {
		startTower()
	}()
	err := waitForServer("127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	defer stopServer()

	assert.Equal(t, "server 1", get("http://127.0.0.1:8000/"))
	assert.Equal(t, "server 1", get("http://127.0.0.1:5000/"))

	exec.Command("cp", "test/servers/server2.go", "test/server1.go").Run()
	defer exec.Command("git", "checkout", "test").Run()
	assert.Equal(t, "server 2", get("http://127.0.0.1:8000/"))
}

func get(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	b_body, _ := ioutil.ReadAll(resp.Body)
	return string(b_body)
}
