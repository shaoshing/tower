package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "server 1")
}

func Error(w http.ResponseWriter, req *http.Request) {
	var paths []string
	paths[0] = "index out of range"
}

func main() {
	appPort := flag.String("p", "5000", "port of your app.")
	flag.Parse()

	http.HandleFunc("/panic", Panic)
	http.HandleFunc("/error", Error)
	http.HandleFunc("/exit", Exit)
	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe(":"+*appPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func Exit(w http.ResponseWriter, req *http.Request) {
	os.Exit(0)
}

func Panic(w http.ResponseWriter, req *http.Request) {
	panic(errors.New("Panic !!"))
}
