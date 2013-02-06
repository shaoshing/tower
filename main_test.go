package main

import (
	"errors"
	"github.com/bmizerany/assert"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCmd(t *testing.T) {
	out, _ := exec.Command("go", "build", "-o", "tmp/tower", "main.go").CombinedOutput()
	if len(out) > 0 {
		panic(errors.New("Could not build server: " + string(out)))
	}

	cmd := exec.Command("tmp/tower", "test/server.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	defer cmd.Process.Kill()

	time.Sleep(5 * time.Second)

	resp, err := http.Get("http://127.0.0.1:8000/")
	if err != nil {
		panic(err)
	}
	b_body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "hello, world!", string(b_body))
}
