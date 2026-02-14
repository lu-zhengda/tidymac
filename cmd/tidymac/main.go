package main

import (
	"fmt"
	"os"

	"github.com/zhengda-lu/tidymac/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
