package main

import (
	"fmt"
	"github.com/ory/x/cmdx"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		cmdx.Fatalf("Expects exactly one input parameter")
	}
	err := filepath.Walk(args[1], func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, "vendor") {
			return nil
		}

		if filepath.Ext(path) == ".go" {
			if p, err := filepath.Abs(filepath.Join(args[1], path)); err != nil {
				return err
			} else {
				fmt.Println(p)
			}
		}

		return nil
	})

	cmdx.Must(err, "%s", err)
}
