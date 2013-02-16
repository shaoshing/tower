package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func dialAddress(address string, timeOut int) (err error) {
	for {
		select {
		case <-time.After(1 * time.Second):
			_, err = net.Dial("tcp", address)
			if err == nil {
				return
			}
		case <-time.After(5 * time.Second):
			fmt.Println("== Waiting for " + address)
		case <-time.After(time.Duration(timeOut) * time.Second):
			return errors.New("Time out")
		}
	}
	return
}

func mustSuccess(err error) {
	if err != nil {
		panic(err)
	}
}
