package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ory/x/watcherx"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "expected 1 comand line argument but got %d\n", len(os.Args)-1)
		os.Exit(1)
	}
	c := make(chan watcherx.Event)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := watcherx.NewFileWatcher(ctx, os.Args[1], c)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "could not initialize file watcher: %+v\n", err)
		os.Exit(1)
	}
	fmt.Printf("watching file %s\n", os.Args[1])
	defer cancel()
	for {
		select {
		case e := <-c:
			var data []byte
			if e.Error == nil {
				data, err = ioutil.ReadAll(e.Data)
			}
			fmt.Printf("got event:\nData: %s,\nError: %+v,\nSrc: %s\n", data, e.Error, e.Src)
		}
	}
}
