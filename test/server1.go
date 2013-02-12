package main

import (
	"errors"
	// "github.com/shaoshing/tower/page"
	"io"
	"log"
	"net/http"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "server 1")
}

func Panic(w http.ResponseWriter, req *http.Request) {
	panic(errors.New("Panic !!"))
}

func Error(w http.ResponseWriter, req *http.Request) {
	var paths []string
	paths[0] = "index out of range"
}

func main() {
	http.HandleFunc("/panic", Panic)
	http.HandleFunc("/error", Error)
	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
